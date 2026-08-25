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
var tenantprofilelog = logf.Log.WithName("tenantprofile-resource")

// SetupTenantProfileWebhookWithManager registers the webhook for TenantProfile in the manager.
func SetupTenantProfileWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &tenancyv1alpha1.TenantProfile{}).
		WithValidator(&TenantProfileCustomValidator{authz: newAuthorizer(mgr)}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-tenancy-opendatahub-io-v1alpha1-tenantprofile,mutating=false,failurePolicy=fail,sideEffects=None,groups=tenancy.opendatahub.io,resources=tenantprofiles,verbs=create;update,versions=v1alpha1,name=vtenantprofile-v1alpha1.kb.io,admissionReviewVersions=v1

// TenantProfileCustomValidator struct is responsible for validating the TenantProfile resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type TenantProfileCustomValidator struct {
	authz *authorizer
}

// ValidateCreate requires a 1:1 name/tenant and ancestor-admin authority.
func (v *TenantProfileCustomValidator) ValidateCreate(ctx context.Context, obj *tenancyv1alpha1.TenantProfile) (admission.Warnings, error) {
	tenantprofilelog.Info("Validation for TenantProfile upon creation", "name", obj.GetName())
	if obj.Name != obj.Spec.Tenant {
		return nil, fmt.Errorf("TenantProfile name %q must equal spec.tenant %q", obj.Name, obj.Spec.Tenant)
	}
	return nil, v.authz.requireAncestorAdmin(ctx, obj.Spec.Tenant, fmt.Sprintf("create profile for tenant %q", obj.Spec.Tenant))
}

// ValidateUpdate enforces tenant immutability. Changing spec.admins requires
// ancestor-admin authority; changing other config requires self-admin authority.
func (v *TenantProfileCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *tenancyv1alpha1.TenantProfile) (admission.Warnings, error) {
	tenantprofilelog.Info("Validation for TenantProfile upon update", "name", newObj.GetName())
	if oldObj.Spec.Tenant != newObj.Spec.Tenant {
		return nil, fmt.Errorf("spec.tenant is immutable: cannot change from %q to %q", oldObj.Spec.Tenant, newObj.Spec.Tenant)
	}
	if !reflect.DeepEqual(oldObj.Spec.Admins, newObj.Spec.Admins) {
		return nil, v.authz.requireAncestorAdmin(ctx, newObj.Spec.Tenant, fmt.Sprintf("modify admins of tenant %q", newObj.Spec.Tenant))
	}
	return nil, v.authz.requireSelfAdmin(ctx, newObj.Spec.Tenant, fmt.Sprintf("modify profile of tenant %q", newObj.Spec.Tenant))
}

// ValidateDelete is not enforced in this PoC.
func (v *TenantProfileCustomValidator) ValidateDelete(_ context.Context, obj *tenancyv1alpha1.TenantProfile) (admission.Warnings, error) {
	return nil, nil
}
