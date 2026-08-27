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

package platformtenant

import (
	"testing"

	. "github.com/onsi/gomega"
	frameworkapi "github.com/opendatahub-io/odh-platform-utilities/framework/api"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

func TestPlatformTenantReconcile_RootTenant(t *testing.T) {
	g := NewWithT(t)
	const resourceName = "pt-test"
	typeNamespacedName := types.NamespacedName{Name: resourceName}

	resource := &tenancyv1alpha1.PlatformTenant{
		ObjectMeta: metav1.ObjectMeta{Name: resourceName},
		Spec:       tenancyv1alpha1.PlatformTenantSpec{DisplayName: "PT Test"},
	}
	g.Expect(k8sClient.Create(ctx, resource)).To(Succeed())
	t.Cleanup(func() {
		pt := &tenancyv1alpha1.PlatformTenant{ObjectMeta: metav1.ObjectMeta{Name: resourceName}}
		g.Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, pt))).To(Succeed())
		profile := &tenancyv1alpha1.TenantProfile{ObjectMeta: metav1.ObjectMeta{Name: resourceName}}
		g.Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, profile))).To(Succeed())
	})

	pt := &tenancyv1alpha1.PlatformTenant{}
	g.Eventually(func() string {
		g.Expect(k8sClient.Get(ctx, typeNamespacedName, pt)).To(Succeed())
		return pt.Status.Root
	}).Should(Equal(resourceName), "status.root should equal itself for a root tenant")

	profile := &tenancyv1alpha1.TenantProfile{}
	g.Eventually(func() error {
		return k8sClient.Get(ctx, typeNamespacedName, profile)
	}).Should(Succeed(), "a restrictive TenantProfile should be auto-created")

	g.Expect(profile.Spec.Tenant).To(Equal(resourceName))
	g.Expect(profile.Spec.Defaults.MaxProjects).To(Equal(int32(0)))
	g.Expect(profile.Spec.Defaults.NetworkIsolation).To(Equal("tenant"))
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
