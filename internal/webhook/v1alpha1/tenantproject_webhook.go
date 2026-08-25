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
var tenantprojectlog = logf.Log.WithName("tenantproject-resource")

// SetupTenantProjectWebhookWithManager registers the webhook for TenantProject in the manager.
func SetupTenantProjectWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &tenancyv1alpha1.TenantProject{}).
		WithValidator(&TenantProjectCustomValidator{authz: newAuthorizer(mgr)}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-tenancy-opendatahub-io-v1alpha1-tenantproject,mutating=false,failurePolicy=fail,sideEffects=None,groups=tenancy.opendatahub.io,resources=tenantprojects,verbs=create;update,versions=v1alpha1,name=vtenantproject-v1alpha1.kb.io,admissionReviewVersions=v1

// TenantProjectCustomValidator struct is responsible for validating the TenantProject resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type TenantProjectCustomValidator struct {
	authz *authorizer
}

// ValidateCreate requires ancestor-admin authority over the owning tenant and
// enforces the tenant's maxProjects cap.
func (v *TenantProjectCustomValidator) ValidateCreate(ctx context.Context, obj *tenancyv1alpha1.TenantProject) (admission.Warnings, error) {
	tenantprojectlog.Info("Validation for TenantProject upon creation", "name", obj.GetName())
	if err := v.authz.requireAncestorAdmin(ctx, obj.Spec.Tenant, fmt.Sprintf("create project in tenant %q", obj.Spec.Tenant)); err != nil {
		return nil, err
	}
	return nil, v.checkProjectQuota(ctx, obj.Spec.Tenant)
}

// ValidateUpdate enforces tenant immutability and ancestor-admin authority.
func (v *TenantProjectCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *tenancyv1alpha1.TenantProject) (admission.Warnings, error) {
	tenantprojectlog.Info("Validation for TenantProject upon update", "name", newObj.GetName())
	if oldObj.Spec.Tenant != newObj.Spec.Tenant {
		return nil, fmt.Errorf("spec.tenant is immutable: cannot change from %q to %q", oldObj.Spec.Tenant, newObj.Spec.Tenant)
	}
	return nil, v.authz.requireAncestorAdmin(ctx, newObj.Spec.Tenant, fmt.Sprintf("update project in tenant %q", newObj.Spec.Tenant))
}

// ValidateDelete is not enforced in this PoC.
func (v *TenantProjectCustomValidator) ValidateDelete(_ context.Context, obj *tenancyv1alpha1.TenantProject) (admission.Warnings, error) {
	return nil, nil
}

// checkProjectQuota rejects the create if the tenant already owns maxProjects.
func (v *TenantProjectCustomValidator) checkProjectQuota(ctx context.Context, tenant string) error {
	var profile tenancyv1alpha1.TenantProfile
	if err := v.authz.reader.Get(ctx, client.ObjectKey{Name: tenant}, &profile); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("tenant %q has no TenantProfile; cannot create projects", tenant)
		}
		return err
	}
	max := profile.Spec.Defaults.MaxProjects

	var projects tenancyv1alpha1.TenantProjectList
	if err := v.authz.reader.List(ctx, &projects); err != nil {
		return err
	}
	count := int32(0)
	for _, p := range projects.Items {
		if p.Spec.Tenant == tenant {
			count++
		}
	}
	if count >= max {
		return fmt.Errorf("tenant %q has reached its maxProjects limit (%d); raise spec.defaults.maxProjects on its TenantProfile", tenant, max)
	}
	return nil
}
