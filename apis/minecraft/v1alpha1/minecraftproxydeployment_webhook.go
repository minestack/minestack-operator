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
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var minecraftproxydeploymentlog = logf.Log.WithName("minecraftproxydeployment-resource")

func (r *MinecraftProxyDeployment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-minecraft-minestack-io-v1alpha1-minecraftproxydeployment,mutating=true,failurePolicy=fail,sideEffects=None,groups=minecraft.minestack.io,resources=minecraftproxydeployments,verbs=create;update,versions=v1alpha1,name=mminecraftproxydeployment.v1alpha1.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &MinecraftProxyDeployment{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *MinecraftProxyDeployment) Default() {
	minecraftproxydeploymentlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:path=/validate-minecraft-minestack-io-v1alpha1-minecraftproxydeployment,mutating=false,failurePolicy=fail,sideEffects=None,groups=minecraft.minestack.io,resources=minecraftproxydeployments,verbs=create;update,versions=v1alpha1,name=vminecraftproxydeployment.v1alpha1.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &MinecraftProxyDeployment{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *MinecraftProxyDeployment) ValidateCreate() error {
	var allErrs field.ErrorList

	minecraftproxydeploymentlog.Info("validate create", "name", r.Name)

	if len(r.Name) > 63 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("name"), r.Name, "Length must be less than 63 characters"))
	}

	if len(r.Spec.Template.Spec.ServerGroups) == 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("template").Child("spec").Child("serverGroups"), 0, "must have at least one server group defined"))
	}

	// TODO: validate Spec.Selector.MatchExpressions is not given
	// TODO: validate Spec.Selector.MatchLabels is given
	// TODO: validate Spec.Template.Metadata.Labels is given
	// TODO: validate Spec.Selector.MatchLabels matches Spec.Template.Metadata.Labels (template labels can have more but selector has to match)

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: r.Kind}, r.Name, nil)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *MinecraftProxyDeployment) ValidateUpdate(old runtime.Object) error {
	var allErrs field.ErrorList

	minecraftproxydeploymentlog.Info("validate update", "name", r.Name)

	oldMPD := old.(*MinecraftServerDeployment)

	// TODO: all validation from create

	if equality.Semantic.DeepEqual(oldMPD.Spec.Template.ObjectMeta.Labels, r.Spec.Template.ObjectMeta.Labels) == false {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec").Child("template").Child("metadata").Child("labels"), "Cannot change template labels"))
	}

	if equality.Semantic.DeepEqual(oldMPD.Spec.Selector, r.Spec.Selector) == false {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec").Child("selector"), "Cannot change selector"))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: r.Kind}, r.Name, nil)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MinecraftProxyDeployment) ValidateDelete() error {
	minecraftproxydeploymentlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
