package tenantmaas

import (
	"context"
	"fmt"
	"github.com/opendatahub-io/odh-platform-utilities/framework/cluster"
	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/types"
	"github.com/opendatahub-io/odh-platform-utilities/framework/resources"
	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const maasNamespace = "ai-tenants"

var maasGVK = schema.GroupVersionKind{Group: "maas.opendatahub.io", Version: "v1alpha1", Kind: "AITenant"}

func reconcileMaaS(ctx context.Context, rr *types.ReconciliationRequest) error {
	binding := rr.Instance.(*tenancyv1alpha1.TenantMaaS)
	binding.Status.AITenant, binding.Status.Message, binding.Status.Ready = "", "", false
	if !hasMaaS(ctx, rr.Client) {
		return nil
	}
	var tenant tenancyv1alpha1.PlatformTenant
	if err := rr.Client.Get(ctx, client.ObjectKey{Name: binding.Spec.TenantRef.Name}, &tenant); err != nil {
		return fmt.Errorf("get tenant: %w", err)
	}
	if tenant.Spec.Parent != "" {
		binding.Status.Message = "MaaS provisioning is supported only for root tenants"
		return nil
	}
	name := binding.Spec.TenantRef.Name
	if len(name) > 41 {
		binding.Status.Message = "tenant name exceeds the AITenant name limit of 41 characters"
		return nil
	}
	obj := resources.GvkToUnstructured(maasGVK)
	obj.SetNamespace(maasNamespace)
	obj.SetName(name)
	err := rr.Client.Get(ctx, client.ObjectKey{Namespace: maasNamespace, Name: name}, obj)
	if apierrors.IsNotFound(err) {
		if err := controllerutil.SetControllerReference(binding, obj, rr.Client.Scheme()); err != nil {
			return fmt.Errorf("set AITenant owner reference: %w", err)
		}
		// Keep the generated AITenant tied to the hierarchy as well. The
		// TenantMaaS binding remains the controller owner, while this additional
		// owner lets PlatformTenant deletion trigger garbage collection.
		if err := controllerutil.SetOwnerReference(&tenant, obj, rr.Client.Scheme()); err != nil {
			return fmt.Errorf("set tenant owner reference: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("get AITenant %q: %w", name, err)
	} else if !metav1.IsControlledBy(obj, binding) {
		binding.Status.Message = "an existing AITenant with this name is not managed by tenancy"
		return nil
	}
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels["tenancy.opendatahub.io/tenant"] = name
	labels["tenancy.opendatahub.io/managed-by"] = "tenancy-controller"
	obj.SetLabels(labels)
	if err := render(obj, &binding.Spec); err != nil {
		return err
	}
	if apierrors.IsNotFound(err) {
		if err := rr.Client.Create(ctx, obj); err != nil {
			return fmt.Errorf("create AITenant: %w", err)
		}
	} else if err := rr.Client.Update(ctx, obj); err != nil {
		return fmt.Errorf("update AITenant: %w", err)
	}
	binding.Status.AITenant = name
	binding.Status.Ready = ready(obj)
	if !binding.Status.Ready {
		binding.Status.Message = "AITenant is waiting for maas-controller"
	}
	return nil
}

func hasMaaS(ctx context.Context, c client.Client) bool {
	ok, err := cluster.HasCRD(ctx, c, maasGVK)
	return err == nil && ok
}
func render(obj *unstructured.Unstructured, spec *tenancyv1alpha1.TenantMaaSSpec) error {
	if spec.OIDC != nil {
		if err := unstructured.SetNestedField(obj.Object, map[string]any{"issuerUrl": spec.OIDC.IssuerURL, "clientId": spec.OIDC.ClientID}, "spec", "oidc"); err != nil {
			return err
		}
	}
	if spec.Gateway != nil {
		if err := unstructured.SetNestedField(obj.Object, map[string]any{"name": spec.Gateway.Name}, "spec", "gateway"); err != nil {
			return err
		}
	}
	if spec.TLS != nil {
		if err := unstructured.SetNestedField(obj.Object, map[string]any{"certificateRef": map[string]any{"name": spec.TLS.CertificateRef.Name, "namespace": spec.TLS.CertificateRef.Namespace}}, "spec", "tls"); err != nil {
			return err
		}
	}
	if spec.Quotas != nil {
		if err := unstructured.SetNestedField(obj.Object, map[string]any{"maxModels": int64(spec.Quotas.MaxModels), "maxSubscriptions": int64(spec.Quotas.MaxSubscriptions), "maxApiKeys": int64(spec.Quotas.MaxAPIKeys)}, "spec", "resourceQuotas"); err != nil {
			return err
		}
	}
	return nil
}
func ready(obj *unstructured.Unstructured) bool {
	conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if !found {
		return false
	}
	for _, raw := range conditions {
		c, ok := raw.(map[string]any)
		if ok && c["type"] == "Ready" && c["status"] == "True" {
			return true
		}
	}
	return false
}
func maasPredicate() predicate.Predicate {
	return predicate.Funcs{CreateFunc: func(e event.CreateEvent) bool { return true }, UpdateFunc: func(e event.UpdateEvent) bool { return true }, DeleteFunc: func(e event.DeleteEvent) bool { return true }, GenericFunc: func(e event.GenericEvent) bool { return false }}
}
func mapAITenant(_ context.Context, obj client.Object) []reconcile.Request {
	for _, owner := range obj.GetOwnerReferences() {
		if owner.APIVersion == tenancyv1alpha1.GroupVersion.String() && owner.Kind == "TenantMaaS" && owner.Controller != nil && *owner.Controller {
			return []reconcile.Request{{NamespacedName: client.ObjectKey{Name: owner.Name}}}
		}
	}
	return nil
}
