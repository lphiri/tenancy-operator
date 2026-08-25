#!/usr/bin/env bash
#
# End-to-end demo of the tenancy-operator authorization model on a kind cluster.
# Uses kubectl impersonation (--as / --as-group) to act as ordinary users so the
# validating webhook is exercised exactly as it would be in production.
#
# Prereqs: the operator is deployed and running, hack/demo-rbac.yaml is applied.
#
# Users:
#   cluster-admin (kubernetes-admin) - trusted bypass, bootstraps the root tenant
#   alice - delegated admin of research-division (an ancestor of genomics)
#   bob   - a plain user with no tenant authority
#
set -uo pipefail

CTX="${CTX:-kind-tenancy-poc}"
KA="kubectl --context ${CTX}"
ALICE="${KA} --as=alice --as-group=tenancy-users"
BOB="${KA} --as=bob --as-group=tenancy-users"

step() { printf '\n\033[1;34m== %s\033[0m\n' "$*"; }
expect_ok()   { if "$@" >/tmp/demo.out 2>&1; then printf '  ALLOWED (expected): %s\n' "$*"; else printf '  UNEXPECTED DENY: %s\n' "$*"; sed 's/^/    /' /tmp/demo.out; fi; }
expect_deny() { if "$@" >/tmp/demo.out 2>&1; then printf '  UNEXPECTED ALLOW: %s\n' "$*"; else printf '  DENIED (expected): %s\n' "$*"; grep -o 'denied the request:.*' /tmp/demo.out | sed 's/^/    /'; fi; }

apply()  { "$@" apply -f - ; }

step "0. Reset demo state"
${KA} delete platformtenant research-division genomics --ignore-not-found >/dev/null 2>&1
${KA} delete tenantproject genomics-analysis genomics-analysis-2 genomics-analysis-3 --ignore-not-found >/dev/null 2>&1
${KA} apply -f hack/demo-rbac.yaml >/dev/null

step "1. cluster-admin creates the root tenant (trusted bypass)"
expect_ok apply ${KA} <<'EOF'
apiVersion: tenancy.opendatahub.io/v1alpha1
kind: PlatformTenant
metadata: { name: research-division }
spec: { displayName: "Research Division" }
EOF
sleep 2
${KA} get platformtenant research-division
echo "  auto-created restrictive profile:"
${KA} get tenantprofile research-division

step "2. alice (no authority yet) tries to create a child tenant -> DENIED"
expect_deny apply ${ALICE} <<'EOF'
apiVersion: tenancy.opendatahub.io/v1alpha1
kind: PlatformTenant
metadata: { name: genomics }
spec: { displayName: "Genomics", parent: research-division }
EOF

step "3. cluster-admin grants alice admin on research-division"
expect_ok apply ${KA} <<'EOF'
apiVersion: tenancy.opendatahub.io/v1alpha1
kind: TenantProfile
metadata: { name: research-division }
spec:
  tenant: research-division
  admins: [ { kind: User, name: alice } ]
  defaults: { networkIsolation: tenant, maxProjects: 0 }
EOF

step "4. alice (ancestor admin) creates the child tenant genomics -> ALLOWED"
expect_ok apply ${ALICE} <<'EOF'
apiVersion: tenancy.opendatahub.io/v1alpha1
kind: PlatformTenant
metadata: { name: genomics }
spec: { displayName: "Genomics", parent: research-division }
EOF
sleep 2
${KA} get platformtenant

step "5. Changing spec.parent is immutable -> DENIED"
expect_deny apply ${ALICE} <<'EOF'
apiVersion: tenancy.opendatahub.io/v1alpha1
kind: PlatformTenant
metadata: { name: genomics }
spec: { displayName: "Genomics", parent: research-division-2 }
EOF

step "6. alice is ancestor-admin but NOT self-admin of genomics."
echo "  6a. alice edits genomics ADMINS (ancestor authority) -> ALLOWED"
expect_ok apply ${ALICE} <<'EOF'
apiVersion: tenancy.opendatahub.io/v1alpha1
kind: TenantProfile
metadata: { name: genomics }
spec:
  tenant: genomics
  admins: [ { kind: User, name: alice } ]
  defaults: { networkIsolation: tenant, maxProjects: 0 }
EOF
echo "  6b. bob (no authority) raises genomics maxProjects -> DENIED"
expect_deny apply ${BOB} <<'EOF'
apiVersion: tenancy.opendatahub.io/v1alpha1
kind: TenantProfile
metadata: { name: genomics }
spec:
  tenant: genomics
  admins: [ { kind: User, name: alice } ]
  defaults: { networkIsolation: tenant, maxProjects: 5 }
EOF

step "7. alice (now self-admin) raises genomics maxProjects to 2 -> ALLOWED"
expect_ok apply ${ALICE} <<'EOF'
apiVersion: tenancy.opendatahub.io/v1alpha1
kind: TenantProfile
metadata: { name: genomics }
spec:
  tenant: genomics
  admins: [ { kind: User, name: alice } ]
  defaults: { networkIsolation: tenant, maxProjects: 2 }
EOF

step "8. alice creates a TenantProject -> ALLOWED, controller provisions namespace + RBAC + NetworkPolicies"
expect_ok apply ${ALICE} <<'EOF'
apiVersion: tenancy.opendatahub.io/v1alpha1
kind: TenantProject
metadata: { name: genomics-analysis }
spec:
  tenant: genomics
  displayName: "Genomics Analysis"
  users:
    - { kind: User, name: bob, role: edit }
    - { kind: User, name: carol, role: view }
EOF
sleep 2
NS=$(${KA} get tenantproject genomics-analysis -o jsonpath='{.status.namespace}')
echo "  provisioned namespace: ${NS}"
${KA} get ns "${NS}" --show-labels
echo "  rolebindings:"; ${KA} -n "${NS}" get rolebindings
echo "  networkpolicies:"; ${KA} -n "${NS}" get networkpolicies

step "9. maxProjects quota enforcement (limit is 2)"
echo "  9a. second project -> ALLOWED"
expect_ok apply ${ALICE} <<'EOF'
apiVersion: tenancy.opendatahub.io/v1alpha1
kind: TenantProject
metadata: { name: genomics-analysis-2 }
spec: { tenant: genomics, displayName: "Analysis 2" }
EOF
echo "  9b. third project exceeds maxProjects=2 -> DENIED"
expect_deny apply ${ALICE} <<'EOF'
apiVersion: tenancy.opendatahub.io/v1alpha1
kind: TenantProject
metadata: { name: genomics-analysis-3 }
spec: { tenant: genomics, displayName: "Analysis 3" }
EOF

step "Demo complete."
