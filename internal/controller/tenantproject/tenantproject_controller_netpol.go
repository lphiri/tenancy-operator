package tenantproject

import (
	"context"
	"fmt"

	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/types"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

const (
	npDenyIngress = "default-deny-ingress"
	npAllowTenant = "allow-from-tenant"
	npAllowSameNS = "allow-same-namespace"
)

// managedNetworkPolicies is the full set of policy names this controller owns.
var managedNetworkPolicies = []string{npDenyIngress, npAllowTenant, npAllowSameNS}

// ensureNetworkPolicies ensures network policies for the isolation preset and
// removes any managed policy the preset does not require.
//
//	none   -> no policies
//	tenant -> default-deny-ingress + allow-from-tenant
//	strict -> default-deny-ingress + allow-same-namespace
func ensureNetworkPolicies(ctx context.Context, rr *types.ReconciliationRequest) error {
	tp := rr.Instance.(*tenancyv1alpha1.TenantProject)

	state := stateFor(rr)
	if state.namespace == nil {
		return fmt.Errorf("namespace not found in reconciliation state")
	}
	if state.isolation == "" {
		return fmt.Errorf("isolation not found in reconciliation state")
	}
	ns := state.namespace

	desired := desiredNetworkPolicies(tp, ns.Name, state.isolation)

	for _, np := range desired {
		policy := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: np.Name, Namespace: ns.Name}}
		_, err := controllerutil.CreateOrUpdate(ctx, rr.Client, policy, func() error {
			policy.Spec = np.Spec
			return controllerutil.SetControllerReference(ns, policy, rr.Client.Scheme())
		})
		if err != nil {
			return err
		}
	}

	wanted := map[string]bool{}
	for _, np := range desired {
		wanted[np.Name] = true
	}
	for _, name := range managedNetworkPolicies {
		if wanted[name] {
			continue
		}
		stale := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns.Name}}
		if err := client.IgnoreNotFound(rr.Client.Delete(ctx, stale)); err != nil {
			return err
		}
	}

	return nil
}

// desiredNetworkPolicies returns the NetworkPolicies required for the preset.
func desiredNetworkPolicies(tp *tenancyv1alpha1.TenantProject, nsName string, isolation string) []networkingv1.NetworkPolicy {
	switch isolation {
	case "none":
		return nil
	case "strict":
		return []networkingv1.NetworkPolicy{denyIngress(nsName), allowSameNamespace(nsName)}
	default: // "tenant"
		return []networkingv1.NetworkPolicy{denyIngress(nsName), allowFromTenant(nsName, tp.Spec.Tenant)}
	}
}

// denyIngress denies all ingress to pods in the namespace by default.
func denyIngress(nsName string) networkingv1.NetworkPolicy {
	return networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: npDenyIngress, Namespace: nsName},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		},
	}
}

// allowFromTenant permits ingress from any namespace of the same tenant.
func allowFromTenant(nsName, tenant string) networkingv1.NetworkPolicy {
	return networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: npAllowTenant, Namespace: nsName},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{labelTenant: tenant},
					},
				}},
			}},
		},
	}
}

// allowSameNamespace permits ingress only from pods in the same namespace.
func allowSameNamespace(nsName string) networkingv1.NetworkPolicy {
	return networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: npAllowSameNS, Namespace: nsName},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					PodSelector: &metav1.LabelSelector{},
				}},
			}},
		},
	}
}
