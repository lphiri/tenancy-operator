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

package v1alpha1

import (
	"context"
	"fmt"
	"os"
	"strings"

	authenticationv1 "k8s.io/api/authentication/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	tenancyv1alpha1 "github.com/opendatahub-io/tenancy-operator/api/v1alpha1"
)

const (
	maxAncestorWalk   = 20
	defaultOperatorNS = "tenancy-operator-system"
)

// defaultClusterAdminGroups are treated as cluster-admin. Covers kubeadm/kind
// ("kubeadm:cluster-admins") and direct system:masters bindings.
var defaultClusterAdminGroups = []string{"system:masters", "kubeadm:cluster-admins"}

// authorizer answers "may this requester act on this tenant?" using the
// hierarchy and the admins declared on each TenantProfile.
type authorizer struct {
	reader             client.Reader
	operatorNamespace  string
	clusterAdminGroups []string
}

func newAuthorizer(mgr manager.Manager) *authorizer {
	ns := os.Getenv("OPERATOR_NAMESPACE")
	if ns == "" {
		ns = defaultOperatorNS
	}
	groups := defaultClusterAdminGroups
	if env := os.Getenv("CLUSTER_ADMIN_GROUPS"); env != "" {
		groups = strings.Split(env, ",")
	}
	return &authorizer{reader: mgr.GetClient(), operatorNamespace: ns, clusterAdminGroups: groups}
}

func userFromContext(ctx context.Context) (authenticationv1.UserInfo, error) {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return authenticationv1.UserInfo{}, err
	}
	return req.UserInfo, nil
}

// isTrusted bypasses tenant checks for cluster-admins and the operator's own
// service account (which auto-creates restrictive TenantProfiles).
func (a *authorizer) isTrusted(u authenticationv1.UserInfo) bool {
	operatorSAGroup := "system:serviceaccounts:" + a.operatorNamespace
	for _, g := range u.Groups {
		if g == operatorSAGroup {
			return true
		}
		for _, admin := range a.clusterAdminGroups {
			if g == admin {
				return true
			}
		}
	}
	return false
}

// subjectMatches reports whether the admission user satisfies an admin subject.
func subjectMatches(s tenancyv1alpha1.Subject, u authenticationv1.UserInfo) bool {
	switch s.Kind {
	case "User":
		return s.Name == u.Username
	case "Group":
		for _, g := range u.Groups {
			if g == s.Name {
				return true
			}
		}
	case "ServiceAccount":
		return u.Username == fmt.Sprintf("system:serviceaccount:%s:%s", s.Namespace, s.Name)
	}
	return false
}

// selfAdmin reports whether the user is an admin listed on the tenant's own
// TenantProfile. A missing profile means no admins, so no match.
func (a *authorizer) selfAdmin(ctx context.Context, u authenticationv1.UserInfo, tenant string) (bool, error) {
	var p tenancyv1alpha1.TenantProfile
	if err := a.reader.Get(ctx, client.ObjectKey{Name: tenant}, &p); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	for _, s := range p.Spec.Admins {
		if subjectMatches(s, u) {
			return true, nil
		}
	}
	return false, nil
}

// ancestorAdmin walks the tenant and its ancestors, returning true if the user
// is an admin anywhere along the chain.
func (a *authorizer) ancestorAdmin(ctx context.Context, u authenticationv1.UserInfo, tenant string) (bool, error) {
	cur := tenant
	for range maxAncestorWalk {
		if cur == "" {
			return false, nil
		}
		ok, err := a.selfAdmin(ctx, u, cur)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
		var pt tenancyv1alpha1.PlatformTenant
		if err := a.reader.Get(ctx, client.ObjectKey{Name: cur}, &pt); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		cur = pt.Spec.Parent
	}
	return false, nil
}

// requireAncestorAdmin permits the action only if the user is an admin of the
// tenant or any ancestor (or is trusted).
func (a *authorizer) requireAncestorAdmin(ctx context.Context, tenant, action string) error {
	u, err := userFromContext(ctx)
	if err != nil {
		return err
	}
	if a.isTrusted(u) {
		return nil
	}
	ok, err := a.ancestorAdmin(ctx, u, tenant)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("user %q is not an admin of tenant %q or any ancestor; cannot %s", u.Username, tenant, action)
	}
	return nil
}

// requireSelfAdmin permits the action only if the user is an admin listed on the
// tenant's own profile (or is trusted). Ancestors do not inherit config rights.
func (a *authorizer) requireSelfAdmin(ctx context.Context, tenant, action string) error {
	u, err := userFromContext(ctx)
	if err != nil {
		return err
	}
	if a.isTrusted(u) {
		return nil
	}
	ok, err := a.selfAdmin(ctx, u, tenant)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("user %q is not an admin of tenant %q; cannot %s", u.Username, tenant, action)
	}
	return nil
}
