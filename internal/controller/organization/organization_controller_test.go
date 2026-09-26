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

package organization

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	frameworkapi "github.com/opendatahub-io/odh-platform-utilities/framework/api"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

func TestOrganizationReconcile_RootOrganization(t *testing.T) {
	g := NewWithT(t)
	const resourceName = "pt-test"
	typeNamespacedName := types.NamespacedName{Name: resourceName}

	resource := &tenancyv1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{Name: resourceName},
		Spec:       tenancyv1alpha1.OrganizationSpec{},
	}
	g.Expect(k8sClient.Create(ctx, resource)).To(Succeed())
	t.Cleanup(func() {
		pt := &tenancyv1alpha1.Organization{ObjectMeta: metav1.ObjectMeta{Name: resourceName}}
		g.Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, pt))).To(Succeed())
		profile := &tenancyv1alpha1.OrganizationProfile{ObjectMeta: metav1.ObjectMeta{Name: resourceName}}
		g.Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, profile))).To(Succeed())
	})

	pt := &tenancyv1alpha1.Organization{}
	g.Eventually(func() string {
		g.Expect(k8sClient.Get(ctx, typeNamespacedName, pt)).To(Succeed())
		return pt.Status.Root
	}).Should(Equal(resourceName), "status.root should equal itself for a root organization")

	profile := &tenancyv1alpha1.OrganizationProfile{}
	g.Eventually(func() error {
		return k8sClient.Get(ctx, typeNamespacedName, profile)
	}).Should(Succeed(), "a restrictive OrganizationProfile should be auto-created")

	g.Expect(profile.Spec.OrganizationRef.Name).To(Equal(resourceName))
	g.Expect(profile.Spec.Defaults.MaxProjects).To(Equal(int32(0)))
	g.Expect(profile.Spec.Defaults.NetworkIsolation).To(Equal("tenant"))
	g.Expect(profile.Labels).To(HaveKeyWithValue(organizationLabel, resourceName))
	g.Expect(profile.Labels).To(HaveKeyWithValue(organizationRootLabel, resourceName))
	g.Expect(profile.OwnerReferences).To(HaveLen(1))
	g.Expect(profile.OwnerReferences[0].Name).To(Equal(resourceName))

	g.Eventually(func() string {
		g.Expect(k8sClient.Get(ctx, typeNamespacedName, pt)).To(Succeed())
		return pt.Status.Phase
	}).Should(Equal("Ready"))
	g.Expect(pt.Status.Conditions).To(ContainElement(And(
		HaveField("Type", string(frameworkapi.ConditionTypeProvisioningSucceeded)),
		HaveField("Status", metav1.ConditionTrue),
		HaveField("ObservedGeneration", pt.Generation),
	)))
	g.Expect(pt.Status.Conditions).To(ContainElement(And(
		HaveField("Type", string(frameworkapi.ConditionTypeReady)),
		HaveField("Status", metav1.ConditionTrue),
	)))
	g.Expect(pt.Status.ObservedGeneration).To(Equal(pt.Generation))
}

func TestComputeRootWalk_RespectsMaxHierarchyDepth(t *testing.T) {
	g := NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(tenancyv1alpha1.AddToScheme(scheme)).To(Succeed())

	root := &tenancyv1alpha1.Organization{ObjectMeta: metav1.ObjectMeta{Name: "org-root"}}
	parent := &tenancyv1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{Name: "org-parent"},
		Spec:       tenancyv1alpha1.OrganizationSpec{Parent: root.Name},
	}
	withinLimit := &tenancyv1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{Name: "org-within-limit"},
		Spec:       tenancyv1alpha1.OrganizationSpec{Parent: parent.Name},
	}
	overLimit := &tenancyv1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{Name: "org-over-limit"},
		Spec:       tenancyv1alpha1.OrganizationSpec{Parent: withinLimit.Name},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(root, parent, withinLimit, overLimit).Build()

	computedRoot, err := computeRootWalk(context.Background(), c, withinLimit)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(computedRoot).To(Equal(root.Name))

	_, err = computeRootWalk(context.Background(), c, overLimit)
	g.Expect(err).To(MatchError(And(
		ContainSubstring(overLimit.Name),
		ContainSubstring("exceeds max depth"),
	)))
}
