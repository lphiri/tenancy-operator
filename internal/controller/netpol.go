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

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

const (
	labelTenant = "tenancy.opendatahub.io/tenant"
	labelRoot   = "tenancy.opendatahub.io/root"

	npDenyIngress = "default-deny-ingress"
	npAllowTenant = "allow-from-tenant"
	npAllowSameNS = "allow-same-namespace"
)

// managedNetworkPolicies is the full set of policy names this controller owns.
// Any not desired for the current preset are deleted.
var managedNetworkPolicies = []string{npDenyIngress, npAllowTenant, npAllowSameNS}

// ensureNetworkPolicies applies the NetworkPolicies for the isolation preset and
// removes any managed policy that the preset does not require.
//
//	none   -> no policies
//	tenant -> default-deny-ingress + allow-from-tenant
//	strict -> default-deny-ingress + allow-same-namespace
func (r *TenantProjectReconciler) ensureNetworkPolicies(ctx context.Context, tp *tenancyv1alpha1.TenantProject, ns *corev1.Namespace, isolation string) error {
	desired := r.desiredNetworkPolicies(tp, ns, isolation)

	for _, np := range desired {
		policy := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: np.Name, Namespace: ns.Name}}
		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, policy, func() error {
			policy.Spec = np.Spec
			return controllerutil.SetControllerReference(ns, policy, r.Scheme)
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
		if err := client.IgnoreNotFound(r.Delete(ctx, stale)); err != nil {
			return err
		}
	}
	return nil
}

// desiredNetworkPolicies returns the NetworkPolicies required for the preset.
func (r *TenantProjectReconciler) desiredNetworkPolicies(tp *tenancyv1alpha1.TenantProject, ns *corev1.Namespace, isolation string) []networkingv1.NetworkPolicy {
	switch isolation {
	case "none":
		return nil
	case "strict":
		return []networkingv1.NetworkPolicy{denyIngress(ns.Name), allowSameNamespace(ns.Name)}
	default: // "tenant"
		return []networkingv1.NetworkPolicy{denyIngress(ns.Name), allowFromTenant(ns.Name, tp.Spec.Tenant)}
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
