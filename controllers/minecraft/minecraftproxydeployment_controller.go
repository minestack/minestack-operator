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
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/minestack/minestack-operator/pkg/api/v1/pod_util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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

// MinecraftProxyDeploymentReconciler reconciles a MinecraftProxyDeployment object
type MinecraftProxyDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=minecraft.minestack.io,resources=minecraftproxydeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=minecraft.minestack.io,resources=minecraftproxydeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=minecraft.minestack.io,resources=minecraftproxydeployments/finalizers,verbs=update

func (r *MinecraftProxyDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	mpd := &minecraftv1alpha1.MinecraftProxyDeployment{}
	err := r.Get(ctx, req.NamespacedName, mpd)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("mpd-%s", mpd.Name),
			Namespace: mpd.Namespace,
		},
	}

	operation, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		err := controllerutil.SetControllerReference(mpd, svc, r.Scheme)
		if err != nil {
			return err
		}

		svc.Labels = mpd.Spec.Template.Labels
		svc.Labels["minestack.io/type"] = "proxy"
		svc.Labels["minestack.io/deployment"] = mpd.Name

		svc.Spec.Selector = mpd.Spec.Selector.MatchLabels
		svc.Spec.Selector["minestack.io/type"] = "proxy"
		svc.Spec.Selector["minestack.io/deployment"] = mpd.Name

		svc.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyTypeLocal

		if len(svc.Spec.Ports) != 1 {
			svc.Spec.Ports = []corev1.ServicePort{{}}
		}

		svc.Spec.Ports[0].Name = "minecraft"
		svc.Spec.Ports[0].Protocol = corev1.ProtocolTCP
		svc.Spec.Ports[0].Port = 25565
		svc.Spec.Ports[0].TargetPort = intstr.FromString("minecraft")

		svc.Spec.Type = corev1.ServiceTypeLoadBalancer
		svc.Spec.SessionAffinity = corev1.ServiceAffinityNone

		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	if operation != controllerutil.OperationResultNone {
		return ctrl.Result{}, nil
	}

	result, err := r.podLogic(ctx, mpd)
	if err != nil {
		return ctrl.Result{}, err
	}

	if result.IsZero() == false {
		return result, nil
	}

	return ctrl.Result{}, nil
}

func (r *MinecraftProxyDeploymentReconciler) podLogic(ctx context.Context, mpd *minecraftv1alpha1.MinecraftProxyDeployment) (ctrl.Result, error) {
	requestedReplicas := int(mpd.Spec.Replicas)

	controllerHasher := fnv.New32a()
	controllerHasher.Reset()
	printer := spew.ConfigState{
		Indent:         " ",
		SortKeys:       true,
		DisableMethods: true,
		SpewKeys:       true,
	}
	_, err := printer.Fprintf(controllerHasher, "%#v", mpd.Spec.Template)
	if err != nil {
		return ctrl.Result{}, err
	}

	podToCreate, err := r.formPod(mpd)
	_, err = printer.Fprintf(controllerHasher, "%#v", podToCreate)
	if err != nil {
		return ctrl.Result{}, err
	}

	controllerHash := rand.SafeEncodeString(fmt.Sprint(controllerHasher.Sum32()))
	podToCreate.Labels["minestack.io/controller-hash"] = controllerHash

	// map of labels to string for kubectl or other parsing
	// labels.SelectorFromSet(msd.Spec.Selector.MatchLabels).String()

	// labels from string to selector to list
	// selector, err := labels.Parse("")
	// err = r.List(ctx, podList, client.MatchingLabelsSelector{Selector: selector})

	podList := &corev1.PodList{}
	err = r.List(ctx, podList, client.MatchingLabels(mpd.Spec.Selector.MatchLabels))
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
				return ctrl.Result{}, err
			}
		}
	}

	if requestedReplicas > foundReplicas {
		err = r.Create(ctx, podToCreate)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

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

		if pod.Status.Phase == corev1.PodFailed {
			failedPods = append(failedPods, pod)
		}
	}

	if availablePods == int32(requestedReplicas) && len(needsUpdate) > 0 {
		err := r.Delete(ctx, &needsUpdate[0])
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	for _, failedPods := range failedPods {
		err := r.Delete(ctx, &failedPods)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	newStatus := minecraftv1alpha1.MinecraftProxyDeploymentStatus{
		AvailableReplicas: availablePods,
		ReadyReplicas:     readyPods,
		Replicas:          mpd.Spec.Replicas,
		UpdatedReplicas:   upToDatePods,
	}

	if equality.Semantic.DeepEqual(newStatus, mpd.Status) == false {
		mpd.Status = newStatus
		err = r.Status().Update(ctx, mpd)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

func (r *MinecraftProxyDeploymentReconciler) formPod(mpd *minecraftv1alpha1.MinecraftProxyDeployment) (*corev1.Pod, error) {
	var groupEnvs []corev1.EnvVar

	for _, group := range mpd.Spec.Template.Spec.ServerGroups {
		matchLabels := group.Selector.MatchLabels
		matchLabels["minestack.io/type"] = "minecraft"

		groupEnvs = append(groupEnvs,
			corev1.EnvVar{
				Name:  fmt.Sprintf("MINESTACK_SERVER_GROUP_%s", group.Name),
				Value: labels.SelectorFromSet(matchLabels).String(),
			},
		)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("mpd-%s-", mpd.Name),
			Namespace:    mpd.Namespace,
			Labels:       mpd.Spec.Template.Labels,
		},
		Spec: corev1.PodSpec{
			EnableServiceLinks:            pointer.Bool(false),
			TerminationGracePeriodSeconds: pointer.Int64(90),
			RestartPolicy:                 corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:            "minecraft",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Image:           mpd.Spec.Template.Spec.Image,
					Env: append(groupEnvs, []corev1.EnvVar{
						{
							Name:  "MINESTACK_PROXY_NAME",
							Value: mpd.Name,
						},
						{
							Name:  "JVM_HEAP",
							Value: fmt.Sprintf("%d", mpd.Spec.Template.Spec.JVMHeap.Value()),
						},
						{
							Name:  "JAVA_ARGS",
							Value: mpd.Spec.Template.Spec.JavaArgs,
						},
					}...),
					Ports: []corev1.ContainerPort{
						{
							Name:          "minecraft",
							ContainerPort: 25565,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "health",
							ContainerPort: 8081,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Resources: mpd.Spec.Template.Spec.Resources,
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.FromInt(25565),
							},
						},
						InitialDelaySeconds: 30,
						TimeoutSeconds:      5,
						PeriodSeconds:       10,
						SuccessThreshold:    5,
						FailureThreshold:    1,
					},
					VolumeMounts: mpd.Spec.Template.Spec.VolumeMounts,
				},
			},
			Volumes: mpd.Spec.Template.Spec.Volumes,
		},
	}

	pod.Labels["minestack.io/type"] = "proxy"
	pod.Labels["minestack.io/deployment"] = mpd.Name

	err := controllerutil.SetControllerReference(mpd, pod, r.Scheme)
	if err != nil {
		return nil, err
	}

	return pod, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MinecraftProxyDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&minecraftv1alpha1.MinecraftProxyDeployment{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
