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

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var platformtenantlog = logf.Log.WithName("platformtenant-resource")

// SetupPlatformTenantWebhookWithManager registers the webhook for PlatformTenant in the manager.
func SetupPlatformTenantWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &tenancyv1alpha1.PlatformTenant{}).
		WithValidator(&PlatformTenantCustomValidator{authz: newAuthorizer(mgr)}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-tenancy-opendatahub-io-v1alpha1-platformtenant,mutating=false,failurePolicy=fail,sideEffects=None,groups=tenancy.opendatahub.io,resources=platformtenants,verbs=create;update,versions=v1alpha1,name=vplatformtenant-v1alpha1.kb.io,admissionReviewVersions=v1

// PlatformTenantCustomValidator struct is responsible for validating the PlatformTenant resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type PlatformTenantCustomValidator struct {
	authz *authorizer
}

// ValidateCreate requires the requester be an admin of the parent tenant or an
// ancestor. Creating a root tenant (no parent) is reserved for trusted users.
func (v *PlatformTenantCustomValidator) ValidateCreate(ctx context.Context, obj *tenancyv1alpha1.PlatformTenant) (admission.Warnings, error) {
	platformtenantlog.Info("Validation for PlatformTenant upon creation", "name", obj.GetName())
	action := fmt.Sprintf("create tenant %q under parent %q", obj.Name, obj.Spec.Parent)
	return nil, v.authz.requireAncestorAdmin(ctx, obj.Spec.Parent, action)
}

// ValidateUpdate enforces parent immutability and ancestor-admin authority.
func (v *PlatformTenantCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *tenancyv1alpha1.PlatformTenant) (admission.Warnings, error) {
	platformtenantlog.Info("Validation for PlatformTenant upon update", "name", newObj.GetName())
	if oldObj.Spec.Parent != newObj.Spec.Parent {
		return nil, fmt.Errorf("spec.parent is immutable: cannot change from %q to %q", oldObj.Spec.Parent, newObj.Spec.Parent)
	}
	return nil, v.authz.requireAncestorAdmin(ctx, newObj.Name, fmt.Sprintf("update tenant %q", newObj.Name))
}

// ValidateDelete is not enforced: deletion is gated by RBAC and owner-reference
// cascade in this PoC.
func (v *PlatformTenantCustomValidator) ValidateDelete(_ context.Context, obj *tenancyv1alpha1.PlatformTenant) (admission.Warnings, error) {
	return nil, nil
}
