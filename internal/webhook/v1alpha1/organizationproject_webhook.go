/*
Copyright 2026.

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
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var organizationprojectlog = logf.Log.WithName("organizationproject-resource")

// SetupOrganizationProjectWebhookWithManager registers the webhook for OrganizationProject.
func SetupOrganizationProjectWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &tenancyv1alpha1.OrganizationProject{}).
		WithValidator(&OrganizationProjectCustomValidator{authz: newAuthorizer(mgr)}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-organization-opendatahub-io-v1alpha1-organizationproject,mutating=false,failurePolicy=fail,sideEffects=None,groups=organization.opendatahub.io,resources=organizationprojects,verbs=create;update,versions=v1alpha1,name=vorganizationproject-v1alpha1.kb.io,admissionReviewVersions=v1

// OrganizationProjectCustomValidator struct is responsible for validating the OrganizationProject resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type OrganizationProjectCustomValidator struct {
	authz *authorizer
}

// ValidateCreate requires ancestor-admin authority over the owning organization and
// enforces the organization's maxProjects cap.
func (v *OrganizationProjectCustomValidator) ValidateCreate(ctx context.Context, obj *tenancyv1alpha1.OrganizationProject) (admission.Warnings, error) {
	organizationprojectlog.Info("Validation for OrganizationProject upon creation", "name", obj.GetName())
	organization := obj.Spec.OrganizationRef.Name
	if err := v.authz.requireAncestorAdmin(ctx, organization, fmt.Sprintf("create project in organization %q", organization)); err != nil {
		return nil, err
	}
	return nil, v.checkProjectQuota(ctx, organization)
}

// ValidateUpdate enforces organization immutability and ancestor-admin authority.
func (v *OrganizationProjectCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *tenancyv1alpha1.OrganizationProject) (admission.Warnings, error) {
	organizationprojectlog.Info("Validation for OrganizationProject upon update", "name", newObj.GetName())
	oldOrganization := oldObj.Spec.OrganizationRef.Name
	organization := newObj.Spec.OrganizationRef.Name
	if oldOrganization != organization {
		return nil, fmt.Errorf("spec.organizationRef.name is immutable: cannot change from %q to %q", oldOrganization, organization)
	}
	return nil, v.authz.requireAncestorAdmin(ctx, organization, fmt.Sprintf("update project in organization %q", organization))
}

// ValidateDelete is not enforced in this PoC.
func (v *OrganizationProjectCustomValidator) ValidateDelete(_ context.Context, obj *tenancyv1alpha1.OrganizationProject) (admission.Warnings, error) {
	return nil, nil
}

// checkProjectQuota rejects the create if the organization already owns maxProjects.
func (v *OrganizationProjectCustomValidator) checkProjectQuota(ctx context.Context, organization string) error {
	var profile tenancyv1alpha1.OrganizationProfile
	if err := v.authz.reader.Get(ctx, client.ObjectKey{Name: organization}, &profile); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("organization %q has no OrganizationProfile; cannot create projects", organization)
		}
		return err
	}
	max := profile.Spec.Defaults.MaxProjects

	var projects tenancyv1alpha1.OrganizationProjectList
	if err := v.authz.reader.List(ctx, &projects); err != nil {
		return err
	}
	count := int32(0)
	for _, p := range projects.Items {
		if p.Spec.OrganizationRef.Name == organization {
			count++
		}
	}
	if count >= max {
		return fmt.Errorf("organization %q has reached its maxProjects limit (%d); raise spec.defaults.maxProjects on its OrganizationProfile", organization, max)
	}
	return nil
}
