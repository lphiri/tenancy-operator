package v1alpha1

import (
	"context"
	"fmt"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func SetupTenantMaaSWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &tenancyv1alpha1.TenantMaaS{}).
		WithValidator(&TenantMaaSCustomValidator{authz: newAuthorizer(mgr)}).Complete()
}

// +kubebuilder:webhook:path=/validate-tenancy-opendatahub-io-v1alpha1-tenantmaas,mutating=false,failurePolicy=fail,sideEffects=None,groups=tenancy.opendatahub.io,resources=tenantmaases,verbs=create;update;delete,versions=v1alpha1,name=vtenantmaas-v1alpha1.kb.io,admissionReviewVersions=v1
type TenantMaaSCustomValidator struct{ authz *authorizer }

func (v *TenantMaaSCustomValidator) ValidateCreate(ctx context.Context, obj *tenancyv1alpha1.TenantMaaS) (admission.Warnings, error) {
	return nil, v.validate(ctx, obj, "create")
}
func (v *TenantMaaSCustomValidator) ValidateUpdate(ctx context.Context, oldObj, obj *tenancyv1alpha1.TenantMaaS) (admission.Warnings, error) {
	if oldObj.Spec.TenantRef.Name != obj.Spec.TenantRef.Name {
		return nil, fmt.Errorf("spec.tenantRef.name is immutable")
	}
	return nil, v.validate(ctx, obj, "update")
}
func (v *TenantMaaSCustomValidator) ValidateDelete(ctx context.Context, obj *tenancyv1alpha1.TenantMaaS) (admission.Warnings, error) {
	return nil, v.authz.requireSelfAdmin(ctx, obj.Spec.TenantRef.Name, fmt.Sprintf("delete MaaS service for tenant %q", obj.Spec.TenantRef.Name))
}
func (v *TenantMaaSCustomValidator) validate(ctx context.Context, obj *tenancyv1alpha1.TenantMaaS, verb string) error {
	if obj.Name != obj.Spec.TenantRef.Name {
		return fmt.Errorf("TenantMaaS name %q must equal spec.tenantRef.name %q", obj.Name, obj.Spec.TenantRef.Name)
	}
	var tenant tenancyv1alpha1.PlatformTenant
	if err := v.authz.reader.Get(ctx, client.ObjectKey{Name: obj.Spec.TenantRef.Name}, &tenant); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("tenant %q does not exist", obj.Spec.TenantRef.Name)
		}
		return err
	}
	if tenant.Spec.Parent != "" {
		return fmt.Errorf("MaaS provisioning is supported only for root tenants")
	}
	return v.authz.requireSelfAdmin(ctx, obj.Spec.TenantRef.Name, fmt.Sprintf("%s MaaS service for tenant %q", verb, obj.Spec.TenantRef.Name))
}
