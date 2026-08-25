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

package controller

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

// maxHierarchyDepth bounds the parent walk to guard against cycles.
const maxHierarchyDepth = 20

// PlatformTenantReconciler reconciles a PlatformTenant object
type PlatformTenantReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=platformtenants,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=platformtenants/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=tenantprofiles,verbs=get;list;watch;create

// Reconcile computes the tenant's root lineage and ensures a restrictive
// TenantProfile exists for it.
func (r *PlatformTenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var pt tenancyv1alpha1.PlatformTenant
	if err := r.Get(ctx, req.NamespacedName, &pt); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	root, err := r.computeRoot(ctx, &pt)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.ensureProfile(ctx, &pt); err != nil {
		return ctrl.Result{}, err
	}

	if pt.Status.Root != root {
		pt.Status.Root = root
		apimeta.SetStatusCondition(&pt.Status.Conditions, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "Reconciled",
			Message:            "Tenant hierarchy resolved and profile ensured",
			ObservedGeneration: pt.Generation,
		})
		if err := r.Status().Update(ctx, &pt); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("reconciled platform tenant", "name", pt.Name, "root", root)
	}

	return ctrl.Result{}, nil
}

// computeRoot walks up the parent chain and returns the root tenant's name.
func (r *PlatformTenantReconciler) computeRoot(ctx context.Context, pt *tenancyv1alpha1.PlatformTenant) (string, error) {
	cur := pt
	for range maxHierarchyDepth {
		if cur.Spec.Parent == "" {
			return cur.Name, nil
		}
		var parent tenancyv1alpha1.PlatformTenant
		if err := r.Get(ctx, client.ObjectKey{Name: cur.Spec.Parent}, &parent); err != nil {
			return "", err
		}
		cur = &parent
	}
	return "", fmt.Errorf("tenant hierarchy for %q exceeds max depth %d", pt.Name, maxHierarchyDepth)
}

// ensureProfile creates a restrictive TenantProfile for the tenant if none
// exists. It never overwrites an existing profile: the profile is self-managed.
func (r *PlatformTenantReconciler) ensureProfile(ctx context.Context, pt *tenancyv1alpha1.PlatformTenant) error {
	var profile tenancyv1alpha1.TenantProfile
	err := r.Get(ctx, client.ObjectKey{Name: pt.Name}, &profile)
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	profile = tenancyv1alpha1.TenantProfile{
		ObjectMeta: metav1.ObjectMeta{Name: pt.Name},
		Spec: tenancyv1alpha1.TenantProfileSpec{
			Tenant: pt.Name,
			Defaults: tenancyv1alpha1.ProjectDefaults{
				NetworkIsolation: "tenant",
				MaxProjects:      0,
			},
		},
	}
	if err := controllerutil.SetControllerReference(pt, &profile, r.Scheme); err != nil {
		return err
	}
	return r.Create(ctx, &profile)
}

// SetupWithManager sets up the controller with the Manager.
func (r *PlatformTenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tenancyv1alpha1.PlatformTenant{}).
		Owns(&tenancyv1alpha1.TenantProfile{}).
		Named("platformtenant").
		Complete(r)
}
