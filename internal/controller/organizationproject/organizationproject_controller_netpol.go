package organizationproject

import (
	"context"
	"fmt"

	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/types"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

const (
	npDenyIngress       = "default-deny-ingress"
	npAllowOrganization = "allow-from-organization"
	npAllowSameNS       = "allow-same-namespace"
)

// managedNetworkPolicies is the full set of policy names this controller owns.
var managedNetworkPolicies = []string{npDenyIngress, npAllowOrganization, npAllowSameNS}

// ensureNetworkPolicies ensures network policies for the isolation preset and
// removes any managed policy the preset does not require.
//
//	none   -> no policies
//	tenant -> default-deny-ingress + allow-from-organization
//	strict -> default-deny-ingress + allow-same-namespace
func ensureNetworkPolicies(ctx context.Context, rr *types.ReconciliationRequest) error {
	project := rr.Instance.(*tenancyv1alpha1.OrganizationProject)

	state := stateFor(rr)
	if state.namespace == nil {
		return fmt.Errorf("namespace not found in reconciliation state")
	}
	if state.isolation == "" {
		return fmt.Errorf("isolation not found in reconciliation state")
	}
	ns := state.namespace

	desired := desiredNetworkPolicies(project, ns.Name, state.isolation)

	for _, np := range desired {
		existing := &networkingv1.NetworkPolicy{}
		err := rr.Client.Get(ctx, client.ObjectKey{Name: np.Name, Namespace: ns.Name}, existing)
		if err == nil && !metav1.IsControlledBy(existing, ns) {
			return fmt.Errorf("network policy %s/%s already exists and is not managed by this OrganizationProject", ns.Name, np.Name)
		}
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		policy := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: np.Name, Namespace: ns.Name}}
		_, err = controllerutil.CreateOrUpdate(ctx, rr.Client, policy, func() error {
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
		if err := rr.Client.Get(ctx, client.ObjectKey{Name: name, Namespace: ns.Name}, stale); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		if !metav1.IsControlledBy(stale, ns) {
			continue
		}
		if err := rr.Client.Delete(ctx, stale); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

// desiredNetworkPolicies returns the NetworkPolicies required for the preset.
func desiredNetworkPolicies(project *tenancyv1alpha1.OrganizationProject, nsName string, isolation string) []networkingv1.NetworkPolicy {
	switch isolation {
	case "none":
		return nil
	case "strict":
		return []networkingv1.NetworkPolicy{denyIngress(nsName), allowSameNamespace(nsName)}
	default: // "tenant"
		return []networkingv1.NetworkPolicy{denyIngress(nsName), allowFromOrganization(nsName, project.Spec.OrganizationRef.Name)}
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

// allowFromOrganization permits ingress from any namespace of the same organization.
func allowFromOrganization(nsName, organization string) networkingv1.NetworkPolicy {
	return networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: npAllowOrganization, Namespace: nsName},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{labelOrganization: organization},
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
