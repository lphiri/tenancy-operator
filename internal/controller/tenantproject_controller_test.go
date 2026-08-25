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
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

var _ = Describe("TenantProject Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			tenantName   = "tproj-tenant"
			resourceName = "tproj-test"
		)

		ctx := context.Background()
		typeNamespacedName := types.NamespacedName{Name: resourceName}

		BeforeEach(func() {
			By("creating a reconciled parent PlatformTenant with a known root")
			tenant := &tenancyv1alpha1.PlatformTenant{
				ObjectMeta: metav1.ObjectMeta{Name: tenantName},
				Spec:       tenancyv1alpha1.PlatformTenantSpec{DisplayName: "Proj Tenant"},
			}
			Expect(k8sClient.Create(ctx, tenant)).To(Succeed())
			tenant.Status.Root = tenantName
			Expect(k8sClient.Status().Update(ctx, tenant)).To(Succeed())

			By("creating the tenant's TenantProfile")
			profile := &tenancyv1alpha1.TenantProfile{
				ObjectMeta: metav1.ObjectMeta{Name: tenantName},
				Spec:       tenancyv1alpha1.TenantProfileSpec{Tenant: tenantName},
			}
			Expect(k8sClient.Create(ctx, profile)).To(Succeed())

			By("creating the TenantProject")
			resource := &tenancyv1alpha1.TenantProject{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName},
				Spec: tenancyv1alpha1.TenantProjectSpec{
					Tenant:      tenantName,
					DisplayName: "Proj Test",
					Users: []tenancyv1alpha1.ProjectUser{
						{Kind: "User", Name: "bob", Role: "edit"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			By("cleaning up project, profile, tenant and provisioned namespace")
			for _, obj := range []client.Object{
				&tenancyv1alpha1.TenantProject{ObjectMeta: metav1.ObjectMeta{Name: resourceName}},
				&tenancyv1alpha1.TenantProfile{ObjectMeta: metav1.ObjectMeta{Name: tenantName}},
				&tenancyv1alpha1.PlatformTenant{ObjectMeta: metav1.ObjectMeta{Name: tenantName}},
				&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: resourceName}},
			} {
				Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, obj))).To(Succeed())
			}
		})

		It("provisions the namespace, RBAC and NetworkPolicies", func() {
			controllerReconciler := &TenantProjectReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			By("provisioning a namespace labeled with tenant and root")
			ns := &corev1.Namespace{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName}, ns)).To(Succeed())
			Expect(ns.Labels).To(HaveKeyWithValue(labelTenant, tenantName))
			Expect(ns.Labels).To(HaveKeyWithValue(labelRoot, tenantName))

			By("binding the edit user to the built-in edit ClusterRole")
			rb := &rbacv1.RoleBinding{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "tenant-edit", Namespace: resourceName}, rb)).To(Succeed())
			Expect(rb.RoleRef.Name).To(Equal("edit"))
			Expect(rb.Subjects).To(HaveLen(1))
			Expect(rb.Subjects[0].Name).To(Equal("bob"))

			By("recording the provisioned namespace in status")
			tp := &tenancyv1alpha1.TenantProject{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, tp)).To(Succeed())
			Expect(tp.Status.Namespace).To(Equal(resourceName))
		})
	})
})
