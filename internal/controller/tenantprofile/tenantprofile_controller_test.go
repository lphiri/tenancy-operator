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

package tenantprofile

import (
	"testing"

	. "github.com/onsi/gomega"
	frameworkapi "github.com/opendatahub-io/odh-platform-utilities/framework/api"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

func TestTenantProfileReconcile_NoError(t *testing.T) {
	g := NewWithT(t)
	const resourceName = "tprof-test"
	typeNamespacedName := types.NamespacedName{Name: resourceName}

	resource := &tenancyv1alpha1.TenantProfile{
		ObjectMeta: metav1.ObjectMeta{Name: resourceName},
		Spec:       tenancyv1alpha1.TenantProfileSpec{Tenant: resourceName},
	}
	g.Expect(k8sClient.Create(ctx, resource)).To(Succeed())
	t.Cleanup(func() {
		g.Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, resource))).To(Succeed())
	})

	profile := &tenancyv1alpha1.TenantProfile{}
	g.Eventually(func() string {
		g.Expect(k8sClient.Get(ctx, typeNamespacedName, profile)).To(Succeed())
		return profile.Status.Phase
	}).Should(Equal("Ready"))
	g.Expect(profile.Status.Conditions).To(ContainElement(And(
		HaveField("Type", string(frameworkapi.ConditionTypeProvisioningSucceeded)),
		HaveField("Status", metav1.ConditionTrue),
		HaveField("ObservedGeneration", profile.Generation),
	)))
	g.Expect(profile.Status.Conditions).To(ContainElement(And(
		HaveField("Type", string(frameworkapi.ConditionTypeReady)),
		HaveField("Status", metav1.ConditionTrue),
	)))
	g.Expect(profile.Status.ObservedGeneration).To(Equal(profile.Generation))
}
