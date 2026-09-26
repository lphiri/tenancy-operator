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
	"reflect"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var organizationprofilelog = logf.Log.WithName("organizationprofile-resource")

// SetupOrganizationProfileWebhookWithManager registers the webhook for OrganizationProfile.
func SetupOrganizationProfileWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &tenancyv1alpha1.OrganizationProfile{}).
		WithValidator(&OrganizationProfileCustomValidator{authz: newAuthorizer(mgr)}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-organization-opendatahub-io-v1alpha1-organizationprofile,mutating=false,failurePolicy=fail,sideEffects=None,groups=organization.opendatahub.io,resources=organizationprofiles,verbs=create;update,versions=v1alpha1,name=vorganizationprofile-v1alpha1.kb.io,admissionReviewVersions=v1

// OrganizationProfileCustomValidator struct is responsible for validating the OrganizationProfile resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type OrganizationProfileCustomValidator struct {
	authz *authorizer
}

// ValidateCreate requires a 1:1 name/organization and ancestor-admin authority.
func (v *OrganizationProfileCustomValidator) ValidateCreate(ctx context.Context, obj *tenancyv1alpha1.OrganizationProfile) (admission.Warnings, error) {
	organizationprofilelog.Info("Validation for OrganizationProfile upon creation", "name", obj.GetName())
	organization := obj.Spec.OrganizationRef.Name
	if obj.Name != organization {
		return nil, fmt.Errorf("OrganizationProfile name %q must equal spec.organizationRef.name %q", obj.Name, organization)
	}
	return nil, v.authz.requireAncestorAdmin(ctx, organization, fmt.Sprintf("create profile for organization %q", organization))
}

// ValidateUpdate enforces organization immutability. Changing spec.admins requires
// ancestor-admin authority; changing other config requires self-admin authority.
func (v *OrganizationProfileCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *tenancyv1alpha1.OrganizationProfile) (admission.Warnings, error) {
	organizationprofilelog.Info("Validation for OrganizationProfile upon update", "name", newObj.GetName())
	oldOrganization := oldObj.Spec.OrganizationRef.Name
	organization := newObj.Spec.OrganizationRef.Name
	if oldOrganization != organization {
		return nil, fmt.Errorf("spec.organizationRef.name is immutable: cannot change from %q to %q", oldOrganization, organization)
	}
	if !reflect.DeepEqual(oldObj.Spec.Admins, newObj.Spec.Admins) {
		return nil, v.authz.requireAncestorAdmin(ctx, organization, fmt.Sprintf("modify admins of organization %q", organization))
	}
	return nil, v.authz.requireSelfAdmin(ctx, organization, fmt.Sprintf("modify profile of organization %q", organization))
}

// ValidateDelete is not enforced in this PoC.
func (v *OrganizationProfileCustomValidator) ValidateDelete(_ context.Context, obj *tenancyv1alpha1.OrganizationProfile) (admission.Warnings, error) {
	return nil, nil
}
