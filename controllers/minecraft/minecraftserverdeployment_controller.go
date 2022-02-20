/*
Copyright 2022 Ryan Belgrave.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package minecraft

import (
	"context"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/minestack/minestack-operator/pkg/api/v1/pod_util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	minecraftv1alpha1 "github.com/minestack/minestack-operator/apis/minecraft/v1alpha1"
)

// MinecraftServerDeploymentReconciler reconciles a MinecraftServerDeployment object
type MinecraftServerDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=minecraft.minestack.io,resources=minecraftserverdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=minecraft.minestack.io,resources=minecraftserverdeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=minecraft.minestack.io,resources=minecraftserverdeployments/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

func (r *MinecraftServerDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	msd := &minecraftv1alpha1.MinecraftServerDeployment{}
	err := r.Get(ctx, req.NamespacedName, msd)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("msd-%s", msd.Name),
			Namespace: msd.Namespace,
		},
	}

	operation, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		err := controllerutil.SetControllerReference(msd, svc, r.Scheme)
		if err != nil {
			return err
		}

		svc.Labels = msd.Spec.Template.Labels
		svc.Labels["minestack.io/type"] = "minecraft"
		svc.Labels["minestack.io/deployment"] = msd.Name

		svc.Spec.Selector = msd.Spec.Selector.MatchLabels
		svc.Spec.Selector["minestack.io/type"] = "minecraft"
		svc.Spec.Selector["minestack.io/deployment"] = msd.Name

		svc.Spec.Ports = []corev1.ServicePort{
			{
				Name:       "minecraft",
				Protocol:   corev1.ProtocolTCP,
				Port:       25565,
				TargetPort: intstr.FromString("minecraft"),
			},
		}

		svc.Spec.ClusterIP = corev1.ClusterIPNone
		svc.Spec.Type = corev1.ServiceTypeClusterIP
		svc.Spec.SessionAffinity = corev1.ServiceAffinityNone

		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	if operation != controllerutil.OperationResultNone {
		return ctrl.Result{}, nil
	}

	var result ctrl.Result
	if msd.Spec.Ordinals {
		result, err = r.podLogicOrdinals(ctx, msd)
	} else {
		result, err = r.podLogicGenerateName(ctx, msd)
	}

	if err != nil {
		return ctrl.Result{}, err
	}

	if result.IsZero() == false {
		return result, nil
	}

	return ctrl.Result{}, nil
}

func (r *MinecraftServerDeploymentReconciler) setStatus(ctx context.Context, msd *minecraftv1alpha1.MinecraftServerDeployment, controllerHash string, pods []corev1.Pod) (ctrl.Result, error) {
	requestedReplicas := int(msd.Spec.Replicas)

	readyPods := int32(0)
	upToDatePods := int32(0)
	availablePods := int32(0)

	var failedPods []corev1.Pod
	var needsUpdate []corev1.Pod
	for _, pod := range pods {
		if pod.DeletionTimestamp.IsZero() {
			if pod_util.IsPodReady(&pod) {
				readyPods += 1
			}

			if pod_util.IsPodAvailable(&pod, 0, metav1.NewTime(time.Now())) {
				availablePods += 1
			}
		}

		if pod.Labels["minestack.io/controller-hash"] != controllerHash {
			needsUpdate = append(needsUpdate, pod)
		} else {
			upToDatePods += 1
		}

		if pod.Status.Phase == corev1.PodFailed || pod_util.IsAContainerTerminated(&pod) {
			failedPods = append(failedPods, pod)
		}
	}

	if (availablePods == 0 || (availablePods == int32(requestedReplicas))) && len(needsUpdate) > 0 {
		err := r.Delete(ctx, &needsUpdate[0])
		if err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	for _, failedPods := range failedPods {
		err := r.Delete(ctx, &failedPods)
		if err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	newStatus := minecraftv1alpha1.MinecraftServerDeploymentStatus{
		AvailableReplicas: availablePods,
		ReadyReplicas:     readyPods,
		Replicas:          msd.Spec.Replicas,
		UpdatedReplicas:   upToDatePods,
	}

	if equality.Semantic.DeepEqual(newStatus, msd.Status) == false {
		msd.Status = newStatus
		err := r.Status().Update(ctx, msd)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

func (r *MinecraftServerDeploymentReconciler) podLogicOrdinals(ctx context.Context, msd *minecraftv1alpha1.MinecraftServerDeployment) (ctrl.Result, error) {
	rLog := log.FromContext(ctx)
	requestedReplicas := int(msd.Spec.Replicas)

	controllerHasher := fnv.New32a()
	controllerHasher.Reset()
	printer := spew.ConfigState{
		Indent:         " ",
		SortKeys:       true,
		DisableMethods: true,
		SpewKeys:       true,
	}
	_, err := printer.Fprintf(controllerHasher, "%#v", msd.Spec.Template)
	if err != nil {
		return ctrl.Result{}, err
	}

	podToCreate, err := r.formPod(msd)
	_, err = printer.Fprintf(controllerHasher, "%#v", podToCreate)
	if err != nil {
		return ctrl.Result{}, err
	}

	controllerHash := rand.SafeEncodeString(fmt.Sprint(controllerHasher.Sum32()))
	podToCreate.Labels["minestack.io/controller-hash"] = controllerHash

	podList := &corev1.PodList{}
	err = r.List(ctx, podList, client.MatchingLabels(msd.Spec.Selector.MatchLabels))
	if err != nil {
		return ctrl.Result{}, err
	}
	pods := podList.Items
	foundReplicas := len(pods)

	var overages []*corev1.Pod
	currentOrdinals := make([]*corev1.Pod, msd.Spec.Replicas, msd.Spec.Replicas)
	for _, pod := range pods {
		podNameParts := strings.Split(pod.Name, "-")
		ordinal, err := strconv.Atoi(podNameParts[len(podNameParts)-1])
		if err != nil {
			return ctrl.Result{}, err
		}

		if ordinal >= cap(currentOrdinals) {
			overages = append(overages, pod.DeepCopy())
		} else {
			currentOrdinals[ordinal] = pod.DeepCopy()
		}
	}

	for _, pod := range overages {
		if pod.DeletionTimestamp.IsZero() == false {
			continue
		}

		rLog.Info("Deleting pod", "pod", &pod.Name)
		err := r.Delete(ctx, pod)
		if err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	if requestedReplicas > foundReplicas {
		nextOrdinal := -1

		for index, pod := range currentOrdinals {
			if pod == nil {
				nextOrdinal = index
				break
			}
		}

		if nextOrdinal == -1 {
			return ctrl.Result{}, fmt.Errorf("could not find ordinal to create next pod at")
		}

		podToCreate.Name = fmt.Sprintf("msd-%s-%d", msd.Name, nextOrdinal)

		err = r.Create(ctx, podToCreate)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}
	}

	return r.setStatus(ctx, msd, controllerHash, pods)
}

func (r *MinecraftServerDeploymentReconciler) podLogicGenerateName(ctx context.Context, msd *minecraftv1alpha1.MinecraftServerDeployment) (ctrl.Result, error) {
	requestedReplicas := int(msd.Spec.Replicas)

	controllerHasher := fnv.New32a()
	controllerHasher.Reset()
	printer := spew.ConfigState{
		Indent:         " ",
		SortKeys:       true,
		DisableMethods: true,
		SpewKeys:       true,
	}
	_, err := printer.Fprintf(controllerHasher, "%#v", msd.Spec.Template)
	if err != nil {
		return ctrl.Result{}, err
	}

	podToCreate, err := r.formPod(msd)
	_, err = printer.Fprintf(controllerHasher, "%#v", podToCreate)
	if err != nil {
		return ctrl.Result{}, err
	}

	controllerHash := rand.SafeEncodeString(fmt.Sprint(controllerHasher.Sum32()))
	podToCreate.Labels["minestack.io/controller-hash"] = controllerHash

	podList := &corev1.PodList{}
	err = r.List(ctx, podList, client.MatchingLabels(msd.Spec.Selector.MatchLabels))
	if err != nil {
		return ctrl.Result{}, err
	}
	pods := podList.Items
	foundReplicas := len(pods)

	if foundReplicas > requestedReplicas {

		for i := 0; i < (foundReplicas - requestedReplicas); i++ {
			pod := pods[(foundReplicas-1)-i]
			if pod.DeletionTimestamp.IsZero() == false {
				continue
			}

			err := r.Delete(ctx, &pod)
			if err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
		}
	}

	if requestedReplicas > foundReplicas {
		err = r.Create(ctx, podToCreate)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return r.setStatus(ctx, msd, controllerHash, pods)
}

func (r *MinecraftServerDeploymentReconciler) formPod(msd *minecraftv1alpha1.MinecraftServerDeployment) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: msd.Namespace,
			Labels:    msd.Spec.Template.Labels,
		},
		Spec: corev1.PodSpec{
			EnableServiceLinks:            pointer.Bool(false),
			TerminationGracePeriodSeconds: pointer.Int64(90),
			RestartPolicy:                 corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:            "minecraft",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Image:           msd.Spec.Template.Spec.Image,
					Env: []corev1.EnvVar{
						{
							Name:  "MINESTACK_SERVER_NAME",
							Value: msd.Name,
						},
						{
							Name:  "JVM_HEAP",
							Value: fmt.Sprintf("%d", msd.Spec.Template.Spec.JVMHeap.Value()),
						},
						{
							Name:  "JAVA_ARGS",
							Value: msd.Spec.Template.Spec.JavaArgs,
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "minecraft",
							ContainerPort: 25565,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Resources: msd.Spec.Template.Spec.Resources,
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.FromString("minecraft"),
							},
						},
						InitialDelaySeconds: 30,
						TimeoutSeconds:      5,
						PeriodSeconds:       10,
						SuccessThreshold:    5,
						FailureThreshold:    1,
					},
					VolumeMounts: msd.Spec.Template.Spec.VolumeMounts,
					Lifecycle: &corev1.Lifecycle{
						PreStop: &corev1.LifecycleHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"/bin/bash",
									"-c",
									"/bin/sleep 10;", // Sleep so our endpoint is removed before we get deleted
									"kill -n SIGTERM 1",
								},
							},
						},
					},
				},
			},
			Volumes: msd.Spec.Template.Spec.Volumes,
		},
	}

	// if no ordinals use generate name
	if msd.Spec.Ordinals == false {
		pod.GenerateName = fmt.Sprintf("msd-%s-", msd.Name)
	}

	pod.Labels["minestack.io/type"] = "minecraft"
	pod.Labels["minestack.io/deployment"] = msd.Name

	err := controllerutil.SetControllerReference(msd, pod, r.Scheme)
	if err != nil {
		return nil, err
	}

	return pod, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MinecraftServerDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&minecraftv1alpha1.MinecraftServerDeployment{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
