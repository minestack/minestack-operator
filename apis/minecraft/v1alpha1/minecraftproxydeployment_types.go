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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type MinecraftProxyGroupSelector struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=256
	// +kubebuilder:validation:Pattern:=`^([a-z]+)$`
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	Selector *metav1.LabelSelector `json:"selector"`
}

type MinecraftProxyServer struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength:=1
	Image string `json:"image"`

	// +kubebuilder:validation:Required
	Resources corev1.ResourceRequirements `json:"resources"`

	// +kubebuilder:validation:Required
	JVMHeap resource.Quantity `json:"JVMHeap"`

	// +kubebuilder:validation:Optional
	JavaArgs string `json:"javaArgs"`

	// +kubebuilder:validation:Optional
	// +nullable
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts"`
}

type MinecraftProxySidecar struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength:=1
	Image string `json:"image"`

	// +kubebuilder:validation:Required
	Resources corev1.ResourceRequirements `json:"resources"`
}

type MinecraftProxySpec struct {
	// +kubebuilder:validation:Required
	Server MinecraftProxyServer `json:"server"`

	// +kubebuilder:validation:Required
	Sidecar MinecraftProxySidecar `json:"sidecar"`

	// +kubebuilder:validation:Optional
	// +nullable
	Volumes []corev1.Volume `json:"volumes"`

	// +kubebuilder:validation:Required
	ServerGroups []MinecraftProxyGroupSelector `json:"serverGroups"`
}

type MinecraftProxyTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Required
	Spec MinecraftProxySpec `json:"spec"`
}

// MinecraftProxyDeploymentSpec defines the desired state of MinecraftProxyDeployment
type MinecraftProxyDeploymentSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum:=0
	Replicas int32 `json:"replicas"`

	// +kubebuilder:validation:Required
	Selector *metav1.LabelSelector `json:"selector"`

	// +kubebuilder:validation:Required
	Template MinecraftProxyTemplate `json:"template"`
}

// MinecraftProxyDeploymentStatus defines the observed state of MinecraftProxyDeployment
type MinecraftProxyDeploymentStatus struct {
	// +kubebuilder:validation:Optional
	AvailableReplicas int32 `json:"availableReplicas"`

	// +kubebuilder:validation:Optional
	ReadyReplicas int32 `json:"readyReplicas"`

	// +kubebuilder:validation:Optional
	Replicas int32 `json:"replicas"`

	// +kubebuilder:validation:Optional
	UpdatedReplicas int32 `json:"updatedReplicas"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=mpd
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.readyReplicas"
// +kubebuilder:printcolumn:name="UP-TO-DATE",type="string",JSONPath=".status.updatedReplicas"
// +kubebuilder:printcolumn:name="AVAILABLE",type="string",JSONPath=".status.availableReplicas"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// MinecraftProxyDeployment is the Schema for the minecraftproxydeployments API
type MinecraftProxyDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Required
	Spec MinecraftProxyDeploymentSpec `json:"spec,omitempty"`

	// +kubebuilder:validation:Optional
	Status MinecraftProxyDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MinecraftProxyDeploymentList contains a list of MinecraftProxyDeployment
type MinecraftProxyDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MinecraftProxyDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MinecraftProxyDeployment{}, &MinecraftProxyDeploymentList{})
}
