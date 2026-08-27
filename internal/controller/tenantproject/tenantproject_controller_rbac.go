package tenantproject

import (
	"context"
	"fmt"

	"github.com/opendatahub-io/odh-platform-utilities/framework/controller/types"
	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ensureRBAC ensures RBAC role bindings for project users.
func ensureRBAC(ctx context.Context, rr *types.ReconciliationRequest) error {
	tp := rr.Instance.(*tenancyv1alpha1.TenantProject)

	state := stateFor(rr)
	if state.namespace == nil {
		return fmt.Errorf("namespace not found in reconciliation state")
	}
	ns := state.namespace

	for _, role := range []string{"edit", "view"} {
		subjects := subjectsForRole(tp.Spec.Users, role)
		rb := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "tenant-" + role, Namespace: ns.Name}}

		if len(subjects) == 0 {
			if err := client.IgnoreNotFound(rr.Client.Delete(ctx, rb)); err != nil {
				return err
			}
			continue
		}

		_, err := controllerutil.CreateOrUpdate(ctx, rr.Client, rb, func() error {
			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     role,
			}
			rb.Subjects = subjects
			return controllerutil.SetControllerReference(ns, rb, rr.Client.Scheme())
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
