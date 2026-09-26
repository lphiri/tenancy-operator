package organization

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

const (
	organizationLabel     = "organization.opendatahub.io/name"
	organizationRootLabel = "organization.opendatahub.io/root"
)

// ensureProfile creates a restrictive OrganizationProfile for the organization.
func ensureProfile(ctx context.Context, rr *types.ReconciliationRequest) error {
	organization := rr.Instance.(*tenancyv1alpha1.Organization)

	var profile tenancyv1alpha1.OrganizationProfile
	err := rr.Client.Get(ctx, client.ObjectKey{Name: organization.Name}, &profile)
	if err == nil {
		return nil // Profile exists
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	// Create default restrictive profile
	profile = tenancyv1alpha1.OrganizationProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name: organization.Name,
			Labels: map[string]string{
				organizationLabel:     organization.Name,
				organizationRootLabel: organization.Status.Root,
			},
		},
		Spec: tenancyv1alpha1.OrganizationProfileSpec{
			OrganizationRef: tenancyv1alpha1.OrganizationReference{Name: organization.Name},
			Defaults: tenancyv1alpha1.ProjectDefaults{
				NetworkIsolation: "tenant",
				MaxProjects:      0,
			},
		},
	}

	if err := controllerutil.SetControllerReference(organization, &profile, rr.Client.Scheme()); err != nil {
		return err
	}

	return rr.Client.Create(ctx, &profile)
}

const maxHierarchyDepth = 3

// computeRoot computes the root organization in the hierarchy.
func computeRoot(ctx context.Context, rr *types.ReconciliationRequest) error {
	organization := rr.Instance.(*tenancyv1alpha1.Organization)

	root, err := computeRootWalk(ctx, rr.Client, organization)
	if err != nil {
		return fmt.Errorf("failed to compute root: %w", err)
	}

	if organization.Status.Root != root {
		organization.Status.Root = root
	}

	return nil
}

func computeRootWalk(ctx context.Context, c client.Client, organization *tenancyv1alpha1.Organization) (string, error) {
	cur := organization
	for range maxHierarchyDepth {
		if cur.Spec.Parent == "" {
			return cur.Name, nil
		}
		var parent tenancyv1alpha1.Organization
		if err := c.Get(ctx, client.ObjectKey{Name: cur.Spec.Parent}, &parent); err != nil {
			return "", err
		}
		cur = &parent
	}
	return "", fmt.Errorf("organization hierarchy for %q exceeds max depth %d", organization.Name, maxHierarchyDepth)
}
