package organizationproject

import (
	"context"
	"fmt"
	"time"

	actionerrors "github.com/opendatahub-io/odh-platform-utilities/framework/controller/actions/errors"
	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/types"
	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const reconciliationStateKey = "organization.opendatahub.io/organizationproject-state"

type reconciliationState struct {
	organization *tenancyv1alpha1.Organization
	namespace    *corev1.Namespace
	isolation    string
}

func stateFor(rr *types.ReconciliationRequest) *reconciliationState {
	if rr.Extensions == nil {
		rr.Extensions = make(map[string]any)
	}

	state, ok := rr.Extensions[reconciliationStateKey].(*reconciliationState)
	if !ok {
		state = &reconciliationState{}
		rr.Extensions[reconciliationStateKey] = state
	}

	return state
}

// validateParentOrganization validates the owning organization exists and has been reconciled.
func validateParentOrganization(ctx context.Context, rr *types.ReconciliationRequest) error {
	project := rr.Instance.(*tenancyv1alpha1.OrganizationProject)

	var organization tenancyv1alpha1.Organization
	if err := rr.Client.Get(ctx, client.ObjectKey{Name: project.Spec.OrganizationRef.Name}, &organization); err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	if organization.Status.Root == "" {
		// Match the previous immediate requeue while allowing the framework to
		// stop the remaining actions cleanly.
		return actionerrors.NewStopError("organization %q not yet reconciled (missing root)", project.Spec.OrganizationRef.Name).
			WithRequeueAfter(10 * time.Second)
	}

	stateFor(rr).organization = &organization

	return nil
}

// computeEffectiveIsolation computes the effective network isolation preset.
func computeEffectiveIsolation(ctx context.Context, rr *types.ReconciliationRequest) error {
	project := rr.Instance.(*tenancyv1alpha1.OrganizationProject)

	isolation := effectiveIsolation(ctx, rr.Client, project)

	stateFor(rr).isolation = isolation

	return nil
}

// effectiveIsolation returns the project's isolation preset, falling back to the
// organization default and finally to "tenant".
func effectiveIsolation(ctx context.Context, c client.Client, project *tenancyv1alpha1.OrganizationProject) string {
	if project.Spec.NetworkIsolation != "" {
		return project.Spec.NetworkIsolation
	}
	var profile tenancyv1alpha1.OrganizationProfile
	if err := c.Get(ctx, client.ObjectKey{Name: project.Spec.OrganizationRef.Name}, &profile); err == nil {
		if profile.Spec.Defaults.NetworkIsolation != "" {
			return profile.Spec.Defaults.NetworkIsolation
		}
	}
	return "tenant"
}
