package maasconfiguration

import (
	"testing"

	. "github.com/onsi/gomega"
	organizationv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestMaaSConfigurationDeletionRemovesCleanupFinalizer(t *testing.T) {
	g := NewWithT(t)
	const name = "maas-cleanup"

	configuration := &organizationv1alpha1.MaaSConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: organizationv1alpha1.MaaSConfigurationSpec{
			OrganizationRef: organizationv1alpha1.OrganizationReference{Name: name},
		},
	}
	g.Expect(k8sClient.Create(ctx, configuration)).To(Succeed())

	g.Eventually(func() []string {
		current := &organizationv1alpha1.MaaSConfiguration{}
		g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name}, current)).To(Succeed())
		return current.Finalizers
	}).Should(ContainElement(maasFinalizer))

	current := &organizationv1alpha1.MaaSConfiguration{}
	g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name}, current)).To(Succeed())
	current.Finalizers = []string{maasFinalizer}
	g.Expect(k8sClient.Update(ctx, current)).To(Succeed())

	g.Expect(k8sClient.Delete(ctx, configuration)).To(Succeed())
	g.Eventually(func() bool {
		current := &organizationv1alpha1.MaaSConfiguration{}
		err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, current)
		return apierrors.IsNotFound(err)
	}).Should(BeTrue())
}
