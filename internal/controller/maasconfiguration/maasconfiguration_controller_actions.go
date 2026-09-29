package maasconfiguration

import (
	"context"
	"fmt"

	"github.com/opendatahub-io/odh-platform-utilities/framework/cluster"
	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/conditions"
	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/types"
	"github.com/opendatahub-io/odh-platform-utilities/framework/resources"
	organizationv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	maasNamespace       = "ai-tenants"
	maasFinalizer       = "organization.opendatahub.io/maas-cleanup"
	organizationLabel   = "organization.opendatahub.io/name"
	managedByLabel      = "maas.opendatahub.io/managed-by"
	temporaryManagedBy  = "tenancy-operator"
	maxAITenantNameSize = 41
	nestedNameField     = "name"
	gatewayField        = "gateway"
	tlsField            = "tls"
)

var maasGVK = schema.GroupVersionKind{Group: "maas.opendatahub.io", Version: "v1alpha1", Kind: "AITenant"}

func reconcileMaaS(ctx context.Context, rr *types.ReconciliationRequest) error {
	configuration := rr.Instance.(*organizationv1alpha1.MaaSConfiguration)

	if !configuration.DeletionTimestamp.IsZero() {
		return cleanupMaaS(ctx, rr, configuration)
	}

	if !controllerutil.ContainsFinalizer(configuration, maasFinalizer) {
		controllerutil.AddFinalizer(configuration, maasFinalizer)
		if err := rr.Client.Update(ctx, configuration); err != nil {
			return fmt.Errorf("add MaaS cleanup finalizer: %w", err)
		}
	}

	if !hasMaaS(ctx, rr.Client) {
		markCapabilityUnavailable(rr)
		return nil
	}
	rr.Conditions.MarkTrue(conditionCapabilityManaged, conditions.WithReason("CapabilityManaged"))

	var organization organizationv1alpha1.Organization
	organizationName := configuration.Spec.OrganizationRef.Name
	if err := rr.Client.Get(ctx, client.ObjectKey{Name: organizationName}, &organization); err != nil {
		return fmt.Errorf("get organization %q: %w", organizationName, err)
	}
	if organization.Spec.Parent != "" {
		rr.Conditions.MarkFalse(conditionMaaSReady,
			conditions.WithReason("RootOrganizationRequired"),
			conditions.WithMessage("MaaS is supported only for root organizations"))
		return nil
	}
	if len(configuration.Name) > maxAITenantNameSize {
		rr.Conditions.MarkFalse(conditionMaaSReady,
			conditions.WithReason("NameTooLong"),
			conditions.WithMessage(fmt.Sprintf("organization name exceeds the AITenant name limit of %d characters", maxAITenantNameSize)))
		return nil
	}

	var profile organizationv1alpha1.OrganizationProfile
	if err := rr.Client.Get(ctx, client.ObjectKey{Name: organizationName}, &profile); err != nil {
		return fmt.Errorf("get organization profile %q: %w", organizationName, err)
	}

	obj := resources.GvkToUnstructured(maasGVK)
	obj.SetNamespace(maasNamespace)
	obj.SetName(configuration.Name)
	err := rr.Client.Get(ctx, client.ObjectKey{Namespace: maasNamespace, Name: configuration.Name}, obj)
	created := apierrors.IsNotFound(err)
	if err != nil && !created {
		return fmt.Errorf("get AITenant %q: %w", configuration.Name, err)
	}

	if created {
		if err := controllerutil.SetControllerReference(configuration, obj, rr.Client.Scheme()); err != nil {
			return fmt.Errorf("set AITenant owner reference: %w", err)
		}
		if err := controllerutil.SetOwnerReference(&organization, obj, rr.Client.Scheme()); err != nil {
			return fmt.Errorf("set organization owner reference: %w", err)
		}
	} else if !metav1.IsControlledBy(obj, configuration) {
		rr.Conditions.MarkFalse(conditionMaaSReady,
			conditions.WithReason("ExistingResourceNotOwned"),
			conditions.WithMessage("an existing AITenant with this name is not managed by this MaaSConfiguration"))
		return nil
	}

	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[organizationLabel] = organizationName
	labels[managedByLabel] = temporaryManagedBy
	obj.SetLabels(labels)

	if err := render(obj, &configuration.Spec, &profile); err != nil {
		return err
	}
	if created {
		if err := rr.Client.Create(ctx, obj); err != nil {
			return fmt.Errorf("create AITenant: %w", err)
		}
	} else if err := rr.Client.Update(ctx, obj); err != nil {
		return fmt.Errorf("update AITenant: %w", err)
	}

	if ready(obj) {
		rr.Conditions.MarkTrue(conditionMaaSReady, conditions.WithReason("MaaSReady"))
	} else {
		rr.Conditions.MarkUnknown(conditionMaaSReady,
			conditions.WithReason("WaitingForMaaS"),
			conditions.WithMessage("AITenant is waiting for the MaaS controller"))
	}
	return nil
}

func cleanupMaaS(ctx context.Context, rr *types.ReconciliationRequest, configuration *organizationv1alpha1.MaaSConfiguration) error {
	obj := resources.GvkToUnstructured(maasGVK)
	obj.SetNamespace(maasNamespace)
	obj.SetName(configuration.Name)
	if err := rr.Client.Get(ctx, client.ObjectKey{Namespace: maasNamespace, Name: configuration.Name}, obj); err != nil {
		if !apierrors.IsNotFound(err) && !apiMeta.IsNoMatchError(err) {
			return fmt.Errorf("get generated AITenant for cleanup: %w", err)
		}
	} else if metav1.IsControlledBy(obj, configuration) {
		if err := rr.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete generated AITenant: %w", err)
		}
	}

	controllerutil.RemoveFinalizer(configuration, maasFinalizer)
	if err := rr.Client.Update(ctx, configuration); err != nil {
		return fmt.Errorf("remove MaaS cleanup finalizer: %w", err)
	}
	return nil
}

func markCapabilityUnavailable(rr *types.ReconciliationRequest) {
	rr.Conditions.MarkFalse(conditionCapabilityManaged,
		conditions.WithReason("CapabilityNotManaged"),
		conditions.WithMessage("the MaaS AITenant CRD is not installed"))
	rr.Conditions.MarkUnknown(conditionMaaSReady,
		conditions.WithReason("CapabilityNotManaged"),
		conditions.WithMessage("MaaS provisioning is waiting for the AITenant CRD"))
}

func hasMaaS(ctx context.Context, c client.Client) bool {
	ok, err := cluster.HasCRD(ctx, c, maasGVK)
	return err == nil && ok
}

func render(obj *unstructured.Unstructured, spec *organizationv1alpha1.MaaSConfigurationSpec, profile *organizationv1alpha1.OrganizationProfile) error {
	oidc := map[string]any{}
	hasOIDC := false
	if profile.Spec.Platform != nil && profile.Spec.Platform.OIDC != nil {
		oidc["issuerUrl"] = profile.Spec.Platform.OIDC.IssuerURL
		hasOIDC = true
	}
	if spec.OIDC != nil {
		oidc["clientId"] = specOIDCClientID(spec)
		hasOIDC = true
	}
	if hasOIDC {
		if err := unstructured.SetNestedField(obj.Object, oidc, "spec", "oidc"); err != nil {
			return err
		}
	} else {
		unstructured.RemoveNestedField(obj.Object, "spec", "oidc")
	}
	if profile.Spec.Platform != nil && profile.Spec.Platform.IngressGatewayRef != nil {
		if err := unstructured.SetNestedField(obj.Object, map[string]any{nestedNameField: profile.Spec.Platform.IngressGatewayRef.Name}, "spec", gatewayField); err != nil {
			return err
		}
	} else {
		unstructured.RemoveNestedField(obj.Object, "spec", gatewayField)
	}
	if spec.TLS != nil {
		if err := unstructured.SetNestedField(obj.Object, map[string]any{
			"certificateRef": map[string]any{
				nestedNameField: spec.TLS.CertificateRef.Name,
				"namespace":     spec.TLS.CertificateRef.Namespace,
			},
		}, "spec", tlsField); err != nil {
			return err
		}
	} else {
		unstructured.RemoveNestedField(obj.Object, "spec", tlsField)
	}
	if spec.Quotas != nil {
		if err := unstructured.SetNestedField(obj.Object, map[string]any{
			"maxModels":        int64(spec.Quotas.MaxModels),
			"maxSubscriptions": int64(spec.Quotas.MaxSubscriptions),
			"maxApiKeys":       int64(spec.Quotas.MaxAPIKeys),
		}, "spec", "resourceQuotas"); err != nil {
			return err
		}
	} else {
		unstructured.RemoveNestedField(obj.Object, "spec", "resourceQuotas")
	}
	return nil
}

func specOIDCClientID(spec *organizationv1alpha1.MaaSConfigurationSpec) string {
	if spec.OIDC == nil {
		return ""
	}
	return spec.OIDC.ClientID
}

func ready(obj *unstructured.Unstructured) bool {
	statusConditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if !found {
		return false
	}
	for _, raw := range statusConditions {
		condition, ok := raw.(map[string]any)
		if ok && condition["type"] == "Ready" && condition["status"] == "True" {
			return true
		}
	}
	return false
}

func maasPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc:  func(event.CreateEvent) bool { return true },
		UpdateFunc:  func(event.UpdateEvent) bool { return true },
		DeleteFunc:  func(event.DeleteEvent) bool { return true },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
}

func mapAITenant(_ context.Context, obj client.Object) []reconcile.Request {
	for _, owner := range obj.GetOwnerReferences() {
		if owner.APIVersion == organizationv1alpha1.GroupVersion.String() && owner.Kind == "MaaSConfiguration" && owner.Controller != nil && *owner.Controller {
			return []reconcile.Request{{NamespacedName: client.ObjectKey{Name: owner.Name}}}
		}
	}
	return nil
}
