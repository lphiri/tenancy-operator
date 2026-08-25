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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

var _ = Describe("PlatformTenant Controller", func() {
	Context("When reconciling a root tenant", func() {
		const resourceName = "pt-test"

		ctx := context.Background()
		typeNamespacedName := types.NamespacedName{Name: resourceName}

		BeforeEach(func() {
			By("creating a root PlatformTenant")
			resource := &tenancyv1alpha1.PlatformTenant{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName},
				Spec:       tenancyv1alpha1.PlatformTenantSpec{DisplayName: "PT Test"},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			By("cleaning up the PlatformTenant and its auto-created TenantProfile")
			pt := &tenancyv1alpha1.PlatformTenant{ObjectMeta: metav1.ObjectMeta{Name: resourceName}}
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, pt))).To(Succeed())
			profile := &tenancyv1alpha1.TenantProfile{ObjectMeta: metav1.ObjectMeta{Name: resourceName}}
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, profile))).To(Succeed())
		})

		It("computes the root and creates a restrictive TenantProfile", func() {
			controllerReconciler := &PlatformTenantReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			By("setting status.root to itself for a root tenant")
			pt := &tenancyv1alpha1.PlatformTenant{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, pt)).To(Succeed())
			Expect(pt.Status.Root).To(Equal(resourceName))

			By("auto-creating a restrictive TenantProfile owned by the tenant")
			profile := &tenancyv1alpha1.TenantProfile{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, profile)).To(Succeed())
			Expect(profile.Spec.Tenant).To(Equal(resourceName))
			Expect(profile.Spec.Defaults.MaxProjects).To(Equal(int32(0)))
			Expect(profile.Spec.Defaults.NetworkIsolation).To(Equal("tenant"))
			Expect(profile.OwnerReferences).To(HaveLen(1))
			Expect(profile.OwnerReferences[0].Name).To(Equal(resourceName))
		})
	})
})
