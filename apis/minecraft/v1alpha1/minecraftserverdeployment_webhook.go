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
var minecraftserverdeploymentlog = logf.Log.WithName("minecraftserverdeployment-resource")

func (r *MinecraftServerDeployment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-minecraft-minestack-io-v1alpha1-minecraftserverdeployment,mutating=true,failurePolicy=fail,sideEffects=None,groups=minecraft.minestack.io,resources=minecraftserverdeployments,verbs=create;update,versions=v1alpha1,name=mminecraftserverdeployment.v1alpha1.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &MinecraftServerDeployment{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *MinecraftServerDeployment) Default() {
	minecraftserverdeploymentlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:path=/validate-minecraft-minestack-io-v1alpha1-minecraftserverdeployment,mutating=false,failurePolicy=fail,sideEffects=None,groups=minecraft.minestack.io,resources=minecraftserverdeployments,verbs=create;update,versions=v1alpha1,name=vminecraftserverdeployment.v1alpha1.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &MinecraftServerDeployment{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *MinecraftServerDeployment) ValidateCreate() error {
	var allErrs field.ErrorList

	minecraftserverdeploymentlog.Info("validate create", "name", r.Name)

	if len(r.Name) > 55 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("name"), r.Name, "Length must be less than 55 characters"))
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
func (r *MinecraftServerDeployment) ValidateUpdate(old runtime.Object) error {
	var allErrs field.ErrorList

	minecraftserverdeploymentlog.Info("validate update", "name", r.Name)

	oldMSD := old.(*MinecraftServerDeployment)

	// TODO: all validation from create

	if equality.Semantic.DeepEqual(oldMSD.Spec.Template.ObjectMeta.Labels, r.Spec.Template.ObjectMeta.Labels) == false {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec").Child("template").Child("metadata").Child("labels"), "Cannot change template labels"))
	}

	if equality.Semantic.DeepEqual(oldMSD.Spec.Selector, r.Spec.Selector) == false {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec").Child("selector"), "Cannot change selector"))
	}

	if oldMSD.Spec.Ordinals != r.Spec.Ordinals {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec").Child("ordinals"), "Cannot change ordinals"))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: r.Kind}, r.Name, nil)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MinecraftServerDeployment) ValidateDelete() error {
	minecraftserverdeploymentlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
