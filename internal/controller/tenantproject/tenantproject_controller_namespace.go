package tenantproject

import (
	"context"
	"fmt"

	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/types"
	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureNamespace ensures the project namespace exists with appropriate labels.
func ensureNamespace(ctx context.Context, rr *types.ReconciliationRequest) error {
	tp := rr.Instance.(*tenancyv1alpha1.TenantProject)

	state := stateFor(rr)
	if state.tenant == nil {
		return fmt.Errorf("parent tenant not found in reconciliation state")
	}

	nsName := tp.Name
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}

	_, err := controllerutil.CreateOrUpdate(ctx, rr.Client, ns, func() error {
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}
		ns.Labels[labelTenant] = tp.Spec.Tenant
		ns.Labels[labelRoot] = state.tenant.Status.Root
		return controllerutil.SetControllerReference(tp, ns, rr.Client.Scheme())
	})

	if err != nil {
		return err
	}

	state.namespace = ns

	// Update status
	if tp.Status.Namespace != nsName {
		tp.Status.Namespace = nsName
	}

	return nil
}
