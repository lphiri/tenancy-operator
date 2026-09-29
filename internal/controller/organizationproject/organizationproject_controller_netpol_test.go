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

package organizationproject

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	frameworktypes "github.com/opendatahub-io/odh-platform-utilities/framework/controller/types"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	organizationv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

func TestEnsureNetworkPoliciesDoesNotAdoptForeignPolicy(t *testing.T) {
	g := NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())
	g.Expect(networkingv1.AddToScheme(scheme)).To(Succeed())
	g.Expect(organizationv1alpha1.AddToScheme(scheme)).To(Succeed())

	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: "foreign-policy-test",
		UID:  types.UID("namespace-uid"),
	}}
	controller := true
	foreign := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{
		Name:      npAllowOrganization,
		Namespace: namespace.Name,
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion: "example.com/v1",
			Kind:       "ForeignController",
			Name:       "foreign",
			UID:        types.UID("foreign-uid"),
			Controller: &controller,
		}},
	}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(namespace, foreign).Build()
	rr := &frameworktypes.ReconciliationRequest{
		Client:   c,
		Instance: &organizationv1alpha1.OrganizationProject{Spec: organizationv1alpha1.OrganizationProjectSpec{OrganizationRef: organizationv1alpha1.OrganizationReference{Name: "org"}}},
	}
	stateFor(rr).namespace = namespace
	stateFor(rr).isolation = "tenant"

	err := ensureNetworkPolicies(context.Background(), rr)
	g.Expect(err).To(MatchError(ContainSubstring("is not managed by this OrganizationProject")))

	got := &networkingv1.NetworkPolicy{}
	g.Expect(c.Get(context.Background(), clientObjectKey(namespace.Name, npAllowOrganization), got)).To(Succeed())
	g.Expect(got.OwnerReferences).To(Equal(foreign.OwnerReferences))
}

func clientObjectKey(namespace, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: namespace, Name: name}
}
