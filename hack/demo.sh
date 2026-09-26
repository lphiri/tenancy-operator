#!/usr/bin/env bash
#
# End-to-end demo of the organization authorization model on a kind cluster.
# Uses kubectl impersonation (--as / --as-group) to act as ordinary users so the
# validating webhook is exercised exactly as it would be in production.
#
# Prereqs: the operator is deployed and running, hack/demo-rbac.yaml is applied.
#
# Users:
#   cluster-admin (kubernetes-admin) - trusted bypass, bootstraps the root organization
#   alice - delegated admin of research-division (an ancestor of genomics)
#   bob   - a plain user with no organization authority
#   carol - delegated admin of the proteomics grandchild organization
#
set -uo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
CTX="${CTX:-kind-tenancy-poc}"
DEMO_DIR="${DEMO_DIR:-${SCRIPT_DIR}/demo}"
KA="kubectl --context ${CTX}"
ALICE="${KA} --as=alice --as-group=tenancy-users"
BOB="${KA} --as=bob --as-group=tenancy-users"
CAROL="${KA} --as=carol --as-group=tenancy-users"

step() { printf '\n\033[1;34m== %s\033[0m\n' "$*"; }
expect_ok()   { if "$@" >/tmp/demo.out 2>&1; then printf '  ALLOWED (expected): %s\n' "$*"; else printf '  UNEXPECTED DENY: %s\n' "$*"; sed 's/^/    /' /tmp/demo.out; fi; }
expect_deny() { if "$@" >/tmp/demo.out 2>&1; then printf '  UNEXPECTED ALLOW: %s\n' "$*"; else printf '  DENIED (expected): %s\n' "$*"; grep -o 'denied the request:.*' /tmp/demo.out | sed 's/^/    /'; fi; }
expect_ok_file() { local file="$1"; shift; expect_ok "$@" apply -f "${DEMO_DIR}/${file}"; }
expect_deny_file() { local file="$1"; shift; expect_deny "$@" apply -f "${DEMO_DIR}/${file}"; }

step "0. Reset demo state"
${KA} delete maascfg research-division genomics --ignore-not-found >/dev/null 2>&1
${KA} delete org research-division genomics proteomics --ignore-not-found >/dev/null 2>&1
${KA} delete org depth-04 --ignore-not-found >/dev/null 2>&1
${KA} delete orgproj genomics-analysis genomics-analysis-2 genomics-analysis-3 proteomics-analysis --ignore-not-found >/dev/null 2>&1
${KA} apply -f "${SCRIPT_DIR}/demo-rbac.yaml" >/dev/null

step "1. cluster-admin creates the root organization (trusted bypass)"
expect_ok_file 01-root-organization.yaml ${KA}
sleep 2
${KA} get org research-division
echo "  auto-created restrictive profile:"
${KA} get orgprof research-division

step "2. alice (no authority yet) tries to create a child organization -> DENIED"
expect_deny_file 02-child-organization.yaml ${ALICE}

step "3. cluster-admin grants alice admin on research-division"
expect_ok_file 03-research-division-profile.yaml ${KA}
echo "  shared OIDC and ingress gateway settings:"
${KA} get orgprof research-division -o yaml

step "4. alice (ancestor admin) creates the child organization genomics -> ALLOWED"
expect_ok_file 02-child-organization.yaml ${ALICE}
sleep 2
${KA} get org

step "5. Changing spec.parent is immutable -> DENIED"
expect_deny_file 04-invalid-parent-change.yaml ${ALICE}

step "6. alice is ancestor-admin but NOT self-admin of genomics."
echo "  6a. alice edits genomics ADMINS (ancestor authority) -> ALLOWED"
expect_ok_file 05-genomics-profile-admin.yaml ${ALICE}
echo "  6b. bob (no authority) raises genomics maxProjects -> DENIED"
expect_deny_file 06-genomics-profile-max-projects-5.yaml ${BOB}

step "7. alice (now self-admin) raises genomics maxProjects to 2 -> ALLOWED"
expect_ok_file 07-genomics-profile-max-projects-2.yaml ${ALICE}

step "8. alice creates an OrganizationProject -> ALLOWED, controller provisions namespace + RBAC + NetworkPolicies"
expect_ok_file 08-first-organization-project.yaml ${ALICE}
sleep 2
NS=$(${KA} get orgproj genomics-analysis -o jsonpath='{.status.namespace}')
echo "  provisioned namespace: ${NS}"
${KA} get ns "${NS}" --show-labels
echo "  rolebindings:"; ${KA} -n "${NS}" get rolebindings
echo "  networkpolicies:"; ${KA} -n "${NS}" get networkpolicies

step "9. maxProjects quota enforcement (limit is 2)"
echo "  9a. second project -> ALLOWED"
expect_ok_file 09-second-organization-project.yaml ${ALICE}
echo "  9b. third project exceeds maxProjects=2 -> DENIED"
expect_deny_file 10-third-organization-project-over-quota.yaml ${ALICE}

step "10. alice creates a grandchild organization under genomics -> ALLOWED"
expect_ok_file 11-grandchild-organization.yaml ${ALICE}
sleep 2
${KA} get org proteomics

step "11. alice delegates administration of proteomics to carol"
expect_ok_file 12-grandchild-profile.yaml ${ALICE}
${KA} get orgprof proteomics

step "12. carol uses her own proteomics admin authority -> ALLOWED"
expect_ok_file 13-grandchild-organization-project.yaml ${CAROL}
sleep 2
${KA} get orgproj proteomics-analysis

step "13. hierarchy depth limit"
echo "  research-division, genomics, and proteomics are the allowed three levels."
echo "  Applying an inspectable fourth-level organization."
${KA} wait --for=condition=Ready org/proteomics --timeout=30s >/dev/null
expect_ok_file 14-depth-limit.yaml ${KA}
sleep 2
echo "  proteomics is Ready; depth-04 reports the depth failure:"
${KA} get org proteomics depth-04 -o custom-columns='NAME:.metadata.name,ROOT:.status.root,PHASE:.status.phase'
echo "  fourth-level organization details:"
${KA} get org depth-04 -o yaml

step "14. alice creates MaaSConfiguration for the root organization -> ALLOWED"
expect_ok_file 15-root-maasconfiguration.yaml ${ALICE}
sleep 2
echo "  shared platform settings used by the organization:"
${KA} get orgprof research-division -o yaml
echo "  MaaSConfiguration status (capability is normally unavailable in this PoC cluster):"
${KA} get maascfg research-division -o yaml

step "15. alice creates MaaSConfiguration for child genomics -> DENIED (root-only)"
expect_deny_file 16-child-maasconfiguration.yaml ${ALICE}

step "Demo complete."
