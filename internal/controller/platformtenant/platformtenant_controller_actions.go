package platformtenant

import (
	"context"
	"fmt"

	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

// EnsureProfile creates an action that ensures a restrictive TenantProfile exists for the tenant
func ensureProfile(ctx context.Context, rr *types.ReconciliationRequest) error {
	pt := rr.Instance.(*tenancyv1alpha1.PlatformTenant)

	var profile tenancyv1alpha1.TenantProfile
	err := rr.Client.Get(ctx, client.ObjectKey{Name: pt.Name}, &profile)
	if err == nil {
		return nil // Profile exists
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	// Create default restrictive profile
	profile = tenancyv1alpha1.TenantProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name: pt.Name,
		},
		Spec: tenancyv1alpha1.TenantProfileSpec{
			Tenant: pt.Name,
			Defaults: tenancyv1alpha1.ProjectDefaults{
				NetworkIsolation: "tenant",
				MaxProjects:      0,
			},
		},
	}

	if err := controllerutil.SetControllerReference(pt, &profile, rr.Client.Scheme()); err != nil {
		return err
	}

	return rr.Client.Create(ctx, &profile)
}

const maxHierarchyDepth = 20

// ComputeRoot creates an action that computes the root tenant in the hierarchy
func computeRoot(ctx context.Context, rr *types.ReconciliationRequest) error {
	pt := rr.Instance.(*tenancyv1alpha1.PlatformTenant)

	root, err := computeRootWalk(ctx, rr.Client, pt)
	if err != nil {
		return fmt.Errorf("failed to compute root: %w", err)
	}

	if pt.Status.Root != root {
		pt.Status.Root = root
	}

	return nil
}

func computeRootWalk(ctx context.Context, c client.Client, pt *tenancyv1alpha1.PlatformTenant) (string, error) {
	cur := pt
	for range maxHierarchyDepth {
		if cur.Spec.Parent == "" {
			return cur.Name, nil
		}
		var parent tenancyv1alpha1.PlatformTenant
		if err := c.Get(ctx, client.ObjectKey{Name: cur.Spec.Parent}, &parent); err != nil {
			return "", err
		}
		cur = &parent
	}
	return "", fmt.Errorf("tenant hierarchy for %q exceeds max depth %d", pt.Name, maxHierarchyDepth)
}
