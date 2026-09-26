package organizationproject

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
	project := rr.Instance.(*tenancyv1alpha1.OrganizationProject)

	state := stateFor(rr)
	if state.organization == nil {
		return fmt.Errorf("organization not found in reconciliation state")
	}

	nsName := project.Name
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}

	_, err := controllerutil.CreateOrUpdate(ctx, rr.Client, ns, func() error {
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}
		ns.Labels[labelOrganization] = project.Spec.OrganizationRef.Name
		ns.Labels[labelOrganizationRoot] = state.organization.Status.Root
		return controllerutil.SetControllerReference(project, ns, rr.Client.Scheme())
	})

	if err != nil {
		return err
	}

	state.namespace = ns

	// Update status
	if project.Status.Namespace != nsName {
		project.Status.Namespace = nsName
	}

	return nil
}
