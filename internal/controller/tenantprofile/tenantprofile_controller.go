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
	"context"

	"github.com/opendatahub-io/odh-platform-utilities/framework/api"
	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/conditions"
	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/reconciler"
	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=tenantprofiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=tenantprofiles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=tenantprofiles/finalizers,verbs=update
// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=platformtenants,verbs=get;list;watch
// +kubebuilder:rbac:groups=maas.opendatahub.io,resources=aitenants,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=maas.opendatahub.io,resources=aitenants/status,verbs=get;watch
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch

// NewTenantProfileReconciler creates and registers a TenantProfile controller using the framework reconciler
// TenantProfile is tenant-managed configuration, so controller has minimal logic
func NewTenantProfileReconciler(ctx context.Context, mgr ctrl.Manager) error {
	_, err := reconciler.ReconcilerFor(mgr, &tenancyv1alpha1.TenantProfile{}).
		WatchesGVK(maasGVK,
			reconciler.Dynamic(reconciler.CrdExists(maasGVK)),
			reconciler.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
			reconciler.WithEventMapper(func(_ context.Context, obj client.Object) []reconcile.Request {
				for _, owner := range obj.GetOwnerReferences() {
					if owner.APIVersion == tenancyv1alpha1.GroupVersion.String() && owner.Kind == "PlatformTenant" && owner.Controller != nil && *owner.Controller {
						return []reconcile.Request{{NamespacedName: k8stypes.NamespacedName{Name: owner.Name}}}
					}
				}
				return nil
			}),
		).
		WithAction(reconcileMaaS).
		WithConditions(
			conditions.Dependent(api.ConditionTypeProvisioningSucceeded, conditions.HealthyWhenTrue),
		).
		Build(ctx)

	return err
}
