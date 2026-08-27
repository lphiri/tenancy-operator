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

package tenantproject

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"
	frameworkapi "github.com/opendatahub-io/odh-platform-utilities/framework/api"
	actionerrors "github.com/opendatahub-io/odh-platform-utilities/framework/controller/actions/errors"
	frameworktypes "github.com/opendatahub-io/odh-platform-utilities/framework/controller/types"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

func TestTenantProjectReconcile_ProvisionsNamespace(t *testing.T) {
	g := NewWithT(t)
	const (
		tenantName   = "tproj-tenant"
		resourceName = "tproj-test"
	)
	typeNamespacedName := types.NamespacedName{Name: resourceName}

	tenant := &tenancyv1alpha1.PlatformTenant{
		ObjectMeta: metav1.ObjectMeta{Name: tenantName},
		Spec:       tenancyv1alpha1.PlatformTenantSpec{DisplayName: "Proj Tenant"},
	}
	g.Expect(k8sClient.Create(ctx, tenant)).To(Succeed())
	tenant.Status.Root = tenantName
	g.Expect(k8sClient.Status().Update(ctx, tenant)).To(Succeed())

	profile := &tenancyv1alpha1.TenantProfile{
		ObjectMeta: metav1.ObjectMeta{Name: tenantName},
		Spec:       tenancyv1alpha1.TenantProfileSpec{Tenant: tenantName},
	}
	g.Expect(k8sClient.Create(ctx, profile)).To(Succeed())

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name:        resourceName,
		Annotations: map[string]string{"example.com/preserved": "true"},
	}}
	g.Expect(k8sClient.Create(ctx, ns)).To(Succeed())

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
	g.Expect(k8sClient.Create(ctx, resource)).To(Succeed())

	t.Cleanup(func() {
		for _, obj := range []client.Object{
			&tenancyv1alpha1.TenantProject{ObjectMeta: metav1.ObjectMeta{Name: resourceName}},
			&tenancyv1alpha1.TenantProfile{ObjectMeta: metav1.ObjectMeta{Name: tenantName}},
			&tenancyv1alpha1.PlatformTenant{ObjectMeta: metav1.ObjectMeta{Name: tenantName}},
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: resourceName}},
		} {
			g.Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, obj))).To(Succeed())
		}
	})

	g.Eventually(func() map[string]string {
		g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName}, ns)).To(Succeed())
		return ns.Labels
	}).Should(And(
		HaveKeyWithValue(labelTenant, tenantName),
		HaveKeyWithValue(labelRoot, tenantName),
	), "the adopted namespace should be labeled")
	g.Expect(ns.Labels).To(HaveKeyWithValue(labelTenant, tenantName))
	g.Expect(ns.Labels).To(HaveKeyWithValue(labelRoot, tenantName))
	g.Expect(ns.Annotations).To(HaveKeyWithValue("example.com/preserved", "true"))

	rb := &rbacv1.RoleBinding{}
	g.Eventually(func() error {
		return k8sClient.Get(ctx, types.NamespacedName{Name: "tenant-edit", Namespace: resourceName}, rb)
	}).Should(Succeed(), "the edit user should be bound to the built-in edit ClusterRole")
	g.Expect(rb.RoleRef.Name).To(Equal("edit"))
	g.Expect(rb.Subjects).To(HaveLen(1))
	g.Expect(rb.Subjects[0].Name).To(Equal("bob"))

	tp := &tenancyv1alpha1.TenantProject{}
	g.Eventually(func() string {
		g.Expect(k8sClient.Get(ctx, typeNamespacedName, tp)).To(Succeed())
		return tp.Status.Namespace
	}).Should(Equal(resourceName), "the provisioned namespace should be recorded in status")
	g.Expect(tp.Status.Phase).To(Equal("Ready"))
	g.Expect(tp.Status.Conditions).To(ContainElement(And(
		HaveField("Type", string(frameworkapi.ConditionTypeProvisioningSucceeded)),
		HaveField("Status", metav1.ConditionTrue),
		HaveField("ObservedGeneration", tp.Generation),
	)))
	g.Expect(tp.Status.Conditions).To(ContainElement(And(
		HaveField("Type", string(frameworkapi.ConditionTypeReady)),
		HaveField("Status", metav1.ConditionTrue),
	)))
	g.Expect(tp.Status.ObservedGeneration).To(Equal(tp.Generation))
}

func TestValidateParentTenant_RequeuesUntilRootIsKnown(t *testing.T) {
	g := NewWithT(t)
	const tenantName = "pending-parent-tenant"

	tenant := &tenancyv1alpha1.PlatformTenant{
		ObjectMeta: metav1.ObjectMeta{Name: tenantName},
		Spec:       tenancyv1alpha1.PlatformTenantSpec{DisplayName: "Pending Parent"},
	}
	g.Expect(k8sClient.Create(ctx, tenant)).To(Succeed())
	t.Cleanup(func() {
		g.Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, tenant))).To(Succeed())
	})

	rr := &frameworktypes.ReconciliationRequest{
		Client: k8sClient,
		Instance: &tenancyv1alpha1.TenantProject{
			Spec: tenancyv1alpha1.TenantProjectSpec{Tenant: tenantName},
		},
	}
	err := validateParentTenant(ctx, rr)
	g.Expect(err).To(HaveOccurred())

	var stopErr actionerrors.StopError
	g.Expect(errors.As(err, &stopErr)).To(BeTrue())
	g.Expect(stopErr.RequeueAfter()).To(BeNumerically(">", 0))
}
