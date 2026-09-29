package maasconfiguration

import (
	"context"

	"github.com/opendatahub-io/odh-platform-utilities/framework/api"
	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/conditions"
	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/reconciler"
	organizationv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	conditionCapabilityManaged api.ConditionType = "CapabilityManaged"
	conditionMaaSReady         api.ConditionType = "MaaSReady"
)

// +kubebuilder:rbac:groups=organization.opendatahub.io,resources=maasconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=organization.opendatahub.io,resources=maasconfigurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=organization.opendatahub.io,resources=maasconfigurations/finalizers,verbs=update
// +kubebuilder:rbac:groups=organization.opendatahub.io,resources=organizations,verbs=get;list;watch
// +kubebuilder:rbac:groups=organization.opendatahub.io,resources=organizations/finalizers,verbs=update
// +kubebuilder:rbac:groups=organization.opendatahub.io,resources=organizationprofiles,verbs=get;list;watch
// +kubebuilder:rbac:groups=maas.opendatahub.io,resources=aitenants,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=maas.opendatahub.io,resources=aitenants/status,verbs=get;watch
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch

// NewMaaSConfigurationReconciler registers the temporary, movable MaaS
// implementation. The package is intentionally independent of the core
// organization controllers so it can later move to the MaaS operator.
func NewMaaSConfigurationReconciler(ctx context.Context, mgr ctrl.Manager) error {
	_, err := reconciler.ReconcilerFor(mgr, &organizationv1alpha1.MaaSConfiguration{}).
		WatchesGVK(maasGVK, reconciler.Dynamic(reconciler.CrdExists(maasGVK)),
			reconciler.WithPredicates(maasPredicate()), reconciler.WithEventMapper(mapAITenant)).
		WithReconcilerOpts(reconciler.WithFinalizerName(maasFinalizer)).
		WithAction(reconcileMaaS).
		WithFinalizer(reconcileMaaS).
		WithConditions(
			conditions.Dependent(conditionCapabilityManaged, conditions.HealthyWhenTrue),
			conditions.Dependent(conditionMaaSReady, conditions.HealthyWhenTrue),
			conditions.Dependent(api.ConditionTypeProvisioningSucceeded, conditions.HealthyWhenTrue),
		).
		Build(ctx)
	return err
}
