package tenantprofile

import (
	"context"
	"fmt"

	"github.com/opendatahub-io/odh-platform-utilities/framework/cluster"
	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/types"
	"github.com/opendatahub-io/odh-platform-utilities/framework/resources"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

const (
	maasNamespace    = "ai-tenants"
	maasGroup        = "maas.opendatahub.io"
	maasVersion      = "v1alpha1"
	maasKind         = "AITenant"
	maasManagedLabel = "tenancy.opendatahub.io/managed-by"
	maasManagedValue = "tenancy-controller"
)

var maasGVK = schema.GroupVersionKind{Group: maasGroup, Version: maasVersion, Kind: maasKind}

// reconcileMaaS renders the tenancy intent into the MaaS API. It deliberately
// uses unstructured objects so MaaS remains the owner of its API types and
// reconciliation logic.
func reconcileMaaS(ctx context.Context, rr *types.ReconciliationRequest) error {
	profile := rr.Instance.(*tenancyv1alpha1.TenantProfile)
	profile.Status.MaaS = tenancyv1alpha1.MaaSStatus{}

	if !clusterHasMaaS(ctx, rr.Client) {
		return nil
	}
	service := profile.Spec.Services.MaaS
	if service == nil || !service.Enabled {
		if err := deleteOwnedAITenant(ctx, rr, profile.Spec.Tenant); err != nil {
			return err
		}
		return nil
	}

	var tenant tenancyv1alpha1.PlatformTenant
	if err := rr.Client.Get(ctx, client.ObjectKey{Name: profile.Spec.Tenant}, &tenant); err != nil {
		return fmt.Errorf("get tenant for MaaS provisioning: %w", err)
	}
	if tenant.Spec.Parent != "" {
		profile.Status.MaaS.Message = "MaaS provisioning is supported only for root tenants"
		return nil
	}
	if len(profile.Spec.Tenant) > 41 {
		profile.Status.MaaS.Message = "tenant name exceeds the AITenant name limit of 41 characters"
		return nil
	}

	aitenant := resources.GvkToUnstructured(maasGVK)
	aitenant.SetNamespace(maasNamespace)
	aitenant.SetName(profile.Spec.Tenant)
	err := rr.Client.Get(ctx, client.ObjectKey{Namespace: maasNamespace, Name: profile.Spec.Tenant}, aitenant)
	if apierrors.IsNotFound(err) {
		if err := controllerutil.SetControllerReference(&tenant, aitenant, rr.Client.Scheme()); err != nil {
			return fmt.Errorf("set AITenant owner reference: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("get AITenant %q: %w", profile.Spec.Tenant, err)
	} else if !metav1.IsControlledBy(aitenant, &tenant) {
		profile.Status.MaaS.Message = "an existing AITenant with this name is not managed by tenancy"
		return nil
	}

	labels := aitenant.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["tenancy.opendatahub.io/tenant"] = profile.Spec.Tenant
	labels[maasManagedLabel] = maasManagedValue
	aitenant.SetLabels(labels)
	if err := renderAITenantSpec(aitenant, service); err != nil {
		return err
	}
	if apierrors.IsNotFound(err) {
		if err := rr.Client.Create(ctx, aitenant); err != nil {
			return fmt.Errorf("create AITenant: %w", err)
		}
	} else if err := rr.Client.Update(ctx, aitenant); err != nil {
		return fmt.Errorf("update AITenant: %w", err)
	}

	profile.Status.MaaS.AITenant = aitenant.GetName()
	profile.Status.MaaS.Ready = aitenantReady(aitenant)
	if !profile.Status.MaaS.Ready {
		profile.Status.MaaS.Message = "AITenant is waiting for maas-controller"
	}
	return nil
}

func deleteOwnedAITenant(ctx context.Context, rr *types.ReconciliationRequest, tenantName string) error {
	aitenant := resources.GvkToUnstructured(maasGVK)
	aitenant.SetNamespace(maasNamespace)
	aitenant.SetName(tenantName)
	if err := rr.Client.Get(ctx, client.ObjectKey{Namespace: maasNamespace, Name: tenantName}, aitenant); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("get AITenant for cleanup: %w", err)
	}
	var tenant tenancyv1alpha1.PlatformTenant
	if err := rr.Client.Get(ctx, client.ObjectKey{Name: tenantName}, &tenant); err != nil {
		return fmt.Errorf("get tenant for AITenant cleanup: %w", err)
	}
	if !metav1.IsControlledBy(aitenant, &tenant) {
		return nil
	}
	if err := rr.Client.Delete(ctx, aitenant); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete AITenant: %w", err)
	}
	return nil
}

func clusterHasMaaS(ctx context.Context, c client.Client) bool {
	ok, err := cluster.HasCRD(ctx, c, maasGVK)
	return err == nil && ok
}

func renderAITenantSpec(obj *unstructured.Unstructured, service *tenancyv1alpha1.MaaSService) error {
	if service.OIDC != nil {
		if err := unstructured.SetNestedField(obj.Object, map[string]any{
			"issuerUrl": service.OIDC.IssuerURL,
			"clientId":  service.OIDC.ClientID,
		}, "spec", "oidc"); err != nil {
			return err
		}
	}
	if service.Gateway != nil {
		if err := unstructured.SetNestedField(obj.Object, map[string]any{"name": service.Gateway.Name}, "spec", "gateway"); err != nil {
			return err
		}
	}
	if service.TLS != nil {
		if err := unstructured.SetNestedField(obj.Object, map[string]any{
			"certificateRef": map[string]any{"name": service.TLS.CertificateRef.Name, "namespace": service.TLS.CertificateRef.Namespace},
		}, "spec", "tls"); err != nil {
			return err
		}
	}
	if service.Quotas != nil {
		if err := unstructured.SetNestedField(obj.Object, map[string]any{
			"maxModels":        int64(service.Quotas.MaxModels),
			"maxSubscriptions": int64(service.Quotas.MaxSubscriptions),
			"maxApiKeys":       int64(service.Quotas.MaxAPIKeys),
		}, "spec", "resourceQuotas"); err != nil {
			return err
		}
	}
	return nil
}

func aitenantReady(obj *unstructured.Unstructured) bool {
	conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if !found {
		return false
	}
	for _, raw := range conditions {
		condition, ok := raw.(map[string]any)
		if ok && condition["type"] == "Ready" && condition["status"] == "True" {
			return true
		}
	}
	return false
}
