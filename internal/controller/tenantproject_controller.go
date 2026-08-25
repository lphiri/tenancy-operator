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

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

// TenantProjectReconciler reconciles a TenantProject object
type TenantProjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=tenantprojects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=tenantprojects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=platformtenants,verbs=get;list;watch
// +kubebuilder:rbac:groups=tenancy.opendatahub.io,resources=tenantprofiles,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=bind,resourceNames=edit;view
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

// Reconcile provisions the project Namespace, its RBAC RoleBindings, and its
// NetworkPolicies.
func (r *TenantProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var tp tenancyv1alpha1.TenantProject
	if err := r.Get(ctx, req.NamespacedName, &tp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var tenant tenancyv1alpha1.PlatformTenant
	if err := r.Get(ctx, client.ObjectKey{Name: tp.Spec.Tenant}, &tenant); err != nil {
		return ctrl.Result{}, err
	}
	if tenant.Status.Root == "" {
		// Tenant not reconciled yet; retry once its root is known.
		return ctrl.Result{Requeue: true}, nil
	}

	isolation := r.effectiveIsolation(ctx, &tp)
	nsName := tp.Name

	ns, err := r.ensureNamespace(ctx, &tp, tenant.Status.Root, nsName)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.ensureRoleBindings(ctx, &tp, ns); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.ensureNetworkPolicies(ctx, &tp, ns, isolation); err != nil {
		return ctrl.Result{}, err
	}

	if tp.Status.Namespace != nsName {
		tp.Status.Namespace = nsName
	}
	apimeta.SetStatusCondition(&tp.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Provisioned",
		Message:            "Namespace, RBAC and NetworkPolicies provisioned",
		ObservedGeneration: tp.Generation,
	})
	if err := r.Status().Update(ctx, &tp); err != nil {
		return ctrl.Result{}, err
	}
	log.Info("reconciled tenant project", "name", tp.Name, "namespace", nsName, "isolation", isolation)
	return ctrl.Result{}, nil
}

// effectiveIsolation returns the project's isolation preset, falling back to the
// tenant default and finally to "tenant".
func (r *TenantProjectReconciler) effectiveIsolation(ctx context.Context, tp *tenancyv1alpha1.TenantProject) string {
	if tp.Spec.NetworkIsolation != "" {
		return tp.Spec.NetworkIsolation
	}
	var profile tenancyv1alpha1.TenantProfile
	if err := r.Get(ctx, client.ObjectKey{Name: tp.Spec.Tenant}, &profile); err == nil {
		if profile.Spec.Defaults.NetworkIsolation != "" {
			return profile.Spec.Defaults.NetworkIsolation
		}
	}
	return "tenant"
}

// ensureNamespace creates or updates the project Namespace with tenant labels.
func (r *TenantProjectReconciler) ensureNamespace(ctx context.Context, tp *tenancyv1alpha1.TenantProject, root, nsName string) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ns, func() error {
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}
		ns.Labels[labelTenant] = tp.Spec.Tenant
		ns.Labels[labelRoot] = root
		return controllerutil.SetControllerReference(tp, ns, r.Scheme)
	})
	return ns, err
}

// ensureRoleBindings binds project users to the built-in edit/view ClusterRoles.
func (r *TenantProjectReconciler) ensureRoleBindings(ctx context.Context, tp *tenancyv1alpha1.TenantProject, ns *corev1.Namespace) error {
	for _, role := range []string{"edit", "view"} {
		subjects := subjectsForRole(tp.Spec.Users, role)
		rb := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "tenant-" + role, Namespace: ns.Name}}
		if len(subjects) == 0 {
			if err := client.IgnoreNotFound(r.Delete(ctx, rb)); err != nil {
				return err
			}
			continue
		}
		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, rb, func() error {
			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     role,
			}
			rb.Subjects = subjects
			return controllerutil.SetControllerReference(ns, rb, r.Scheme)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// subjectsForRole returns the RBAC subjects for users granted the given role.
func subjectsForRole(users []tenancyv1alpha1.ProjectUser, role string) []rbacv1.Subject {
	var subjects []rbacv1.Subject
	for _, u := range users {
		if u.Role != role {
			continue
		}
		s := rbacv1.Subject{Kind: u.Kind, Name: u.Name}
		if u.Kind == "ServiceAccount" {
			s.Namespace = u.Namespace
		} else {
			s.APIGroup = rbacv1.GroupName
		}
		subjects = append(subjects, s)
	}
	return subjects
}

// SetupWithManager sets up the controller with the Manager.
func (r *TenantProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tenancyv1alpha1.TenantProject{}).
		Owns(&corev1.Namespace{}).
		Named("tenantproject").
		Complete(r)
}
