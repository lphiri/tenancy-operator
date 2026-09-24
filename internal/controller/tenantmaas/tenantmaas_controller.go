package tenantmaas

import (
	"context"
	"github.com/opendatahub-io/odh-platform-utilities/framework/api"
	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/conditions"
	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/reconciler"
	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=tenantmaases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=tenantmaases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=tenantmaases/finalizers,verbs=update
// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=platformtenants,verbs=get;list;watch
// +kubebuilder:rbac:groups=maas.opendatahub.io,resources=aitenants,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=maas.opendatahub.io,resources=aitenants/status,verbs=get;watch
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch
func NewTenantMaaSReconciler(ctx context.Context, mgr ctrl.Manager) error {
	_, err := reconciler.ReconcilerFor(mgr, &tenancyv1alpha1.TenantMaaS{}).
		WatchesGVK(maasGVK, reconciler.Dynamic(reconciler.CrdExists(maasGVK)),
			reconciler.WithPredicates(maasPredicate()), reconciler.WithEventMapper(mapAITenant)).
		WithAction(reconcileMaaS).
		WithConditions(conditions.Dependent(api.ConditionTypeProvisioningSucceeded, conditions.HealthyWhenTrue)).
		Build(ctx)
	return err
}
