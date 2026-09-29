// +kubebuilder:webhook:path=/validate-organization-opendatahub-io-v1alpha1-maasconfiguration,mutating=false,failurePolicy=fail,sideEffects=None,groups=organization.opendatahub.io,resources=maasconfigurations,verbs=create;update;delete,versions=v1alpha1,name=vmaasconfiguration-v1alpha1.kb.io,admissionReviewVersions=v1
package v1alpha1

import (
	"context"
	"fmt"

	organizationv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupMaaSConfigurationWebhookWithManager registers the temporary MaaS
// webhook. It is kept separate so the implementation can move with the MaaS
// controller later.
func SetupMaaSConfigurationWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &organizationv1alpha1.MaaSConfiguration{}).
		WithValidator(&MaaSConfigurationCustomValidator{authz: newAuthorizer(mgr)}).Complete()
}

type MaaSConfigurationCustomValidator struct{ authz *authorizer }

func (v *MaaSConfigurationCustomValidator) ValidateCreate(ctx context.Context, obj *organizationv1alpha1.MaaSConfiguration) (admission.Warnings, error) {
	return nil, v.validate(ctx, obj, "create")
}

func (v *MaaSConfigurationCustomValidator) ValidateUpdate(ctx context.Context, oldObj, obj *organizationv1alpha1.MaaSConfiguration) (admission.Warnings, error) {
	if oldObj.Spec.OrganizationRef.Name != obj.Spec.OrganizationRef.Name {
		return nil, fmt.Errorf("spec.organizationRef.name is immutable")
	}
	return nil, v.validate(ctx, obj, "update")
}

func (v *MaaSConfigurationCustomValidator) ValidateDelete(ctx context.Context, obj *organizationv1alpha1.MaaSConfiguration) (admission.Warnings, error) {
	return nil, v.authz.requireSelfAdmin(ctx, obj.Spec.OrganizationRef.Name,
		fmt.Sprintf("delete MaaS configuration for organization %q", obj.Spec.OrganizationRef.Name))
}

func (v *MaaSConfigurationCustomValidator) validate(ctx context.Context, obj *organizationv1alpha1.MaaSConfiguration, verb string) error {
	organizationName := obj.Spec.OrganizationRef.Name
	if organizationName == "" {
		return fmt.Errorf("spec.organizationRef.name is required")
	}
	if obj.Name != organizationName {
		return fmt.Errorf("MaaSConfiguration name %q must equal spec.organizationRef.name %q", obj.Name, organizationName)
	}

	var organization organizationv1alpha1.Organization
	if err := v.authz.reader.Get(ctx, client.ObjectKey{Name: organizationName}, &organization); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("organization %q does not exist", organizationName)
		}
		return err
	}
	if organization.Spec.Parent != "" {
		return fmt.Errorf("MaaSConfiguration is supported only for root Organizations")
	}

	var configurations organizationv1alpha1.MaaSConfigurationList
	if err := v.authz.reader.List(ctx, &configurations); err != nil {
		return err
	}
	for _, existing := range configurations.Items {
		if existing.Name != obj.Name && existing.Spec.OrganizationRef.Name == organizationName {
			return fmt.Errorf("organization %q already has MaaSConfiguration %q", organizationName, existing.Name)
		}
	}

	return v.authz.requireSelfAdmin(ctx, organizationName,
		fmt.Sprintf("%s MaaS configuration for organization %q", verb, organizationName))
}
