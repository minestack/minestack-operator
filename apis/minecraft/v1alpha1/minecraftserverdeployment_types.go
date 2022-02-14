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

type MinecraftServerSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength:=1
	Image string `json:"image"`

	// +kubebuilder:validation:Required
	Resources corev1.ResourceRequirements `json:"resources"`

	// +kubebuilder:validation:Required
	JVMHeap resource.Quantity `json:"JVMHeap"`

	// +kubebuilder:validation:Optional
	JavaArgs string `json:"javaArgs"`
}

type MinecraftServerTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Required
	Spec MinecraftServerSpec `json:"spec"`
}

// MinecraftServerDeploymentSpec defines the desired state of MinecraftServerDeployment
type MinecraftServerDeploymentSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum:=1
	Replicas int32 `json:"replicas"`

	// +kubebuilder:validation:Required
	Selector *metav1.LabelSelector `json:"selector"`

	// +kubebuilder:validation:Required
	Template MinecraftServerTemplate `json:"template"`
}

// MinecraftServerDeploymentStatus defines the observed state of MinecraftServerDeployment
type MinecraftServerDeploymentStatus struct {
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
// +kubebuilder:resource:shortName=msd
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.readyReplicas"
// +kubebuilder:printcolumn:name="UP-TO-DATE",type="string",JSONPath=".status.updatedReplicas"
// +kubebuilder:printcolumn:name="AVAILABLE",type="string",JSONPath=".status.availableReplicas"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// MinecraftServerDeployment is the Schema for the minecraftserverdeployments API
type MinecraftServerDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Required
	Spec MinecraftServerDeploymentSpec `json:"spec,omitempty"`

	// +kubebuilder:validation:Optional
	Status MinecraftServerDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MinecraftServerDeploymentList contains a list of MinecraftServerDeployment
type MinecraftServerDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MinecraftServerDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MinecraftServerDeployment{}, &MinecraftServerDeploymentList{})
}
