package tenantproject

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

const reconciliationStateKey = "tenancy.opendatahub.io/tenantproject-state"

type reconciliationState struct {
	tenant    *tenancyv1alpha1.PlatformTenant
	namespace *corev1.Namespace
	isolation string
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

// validateParentTenant validates the parent tenant exists and has been reconciled.
func validateParentTenant(ctx context.Context, rr *types.ReconciliationRequest) error {
	tp := rr.Instance.(*tenancyv1alpha1.TenantProject)

	var tenant tenancyv1alpha1.PlatformTenant
	if err := rr.Client.Get(ctx, client.ObjectKey{Name: tp.Spec.Tenant}, &tenant); err != nil {
		return fmt.Errorf("failed to get parent tenant: %w", err)
	}

	if tenant.Status.Root == "" {
		// Match the previous immediate requeue while allowing the framework to
		// stop the remaining actions cleanly.
		return actionerrors.NewStopError("tenant %q not yet reconciled (missing root)", tp.Spec.Tenant).
			WithRequeueAfter(10 * time.Second)
	}

	stateFor(rr).tenant = &tenant

	return nil
}

// computeEffectiveIsolation computes the effective network isolation preset.
func computeEffectiveIsolation(ctx context.Context, rr *types.ReconciliationRequest) error {
	tp := rr.Instance.(*tenancyv1alpha1.TenantProject)

	isolation := effectiveIsolation(ctx, rr.Client, tp)

	stateFor(rr).isolation = isolation

	return nil
}

// effectiveIsolation returns the project's isolation preset, falling back to the
// tenant default and finally to "tenant".
func effectiveIsolation(ctx context.Context, c client.Client, tp *tenancyv1alpha1.TenantProject) string {
	if tp.Spec.NetworkIsolation != "" {
		return tp.Spec.NetworkIsolation
	}
	var profile tenancyv1alpha1.TenantProfile
	if err := c.Get(ctx, client.ObjectKey{Name: tp.Spec.Tenant}, &profile); err == nil {
		if profile.Spec.Defaults.NetworkIsolation != "" {
			return profile.Spec.Defaults.NetworkIsolation
		}
	}
	return "tenant"
}
