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

package tenantproject

import (
	"context"

	"github.com/opendatahub-io/odh-platform-utilities/framework/api"
	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/conditions"
	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/reconciler"
	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	labelTenant = "tenancy.opendatahub.io/tenant"
	labelRoot   = "tenancy.opendatahub.io/root"
)

// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=tenantprojects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=tenantprojects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=platformtenants,verbs=get;list;watch
// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=tenantprofiles,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=bind,resourceNames=edit;view
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

// NewTenantProjectReconciler creates and registers a TenantProject controller using the framework reconciler
func NewTenantProjectReconciler(ctx context.Context, mgr ctrl.Manager) error {
	_, err := reconciler.ReconcilerFor(mgr, &tenancyv1alpha1.TenantProject{}).
		Owns(&corev1.Namespace{}).
		WithAction(validateParentTenant).
		WithAction(computeEffectiveIsolation).
		WithAction(ensureNamespace).
		WithAction(ensureRBAC).
		WithAction(ensureNetworkPolicies).
		WithConditions(
			conditions.Dependent(api.ConditionTypeProvisioningSucceeded, conditions.HealthyWhenTrue),
		).
		Build(ctx)

	return err
}
