# Delegated MaaS Implementation Plan

This plan applies the approved API naming and temporary MaaS ownership decisions to the tenancy-operator proof of concept.

## Decisions

- Keep the canonical resource names:
  - `Organization`
  - `OrganizationProfile`
  - `OrganizationProject`
  - `MaaSConfiguration`
- Use the API group and version `organization.opendatahub.io/v1alpha1`.
- Expose the following `kubectl` short names:
  - `org`
  - `orgprof`
  - `orgproj`
  - `maascfg`
- Rename `tenantRef` and `tenant` references to `organizationRef`.
- Use `organization.opendatahub.io/*` labels and annotations.
- Keep the MaaS implementation in this repository temporarily, but isolate it so it can later move to a separate MaaS operator.
- Treat the current API as a pre-release proof of concept. Perform a direct API rename; no conversion webhook or compatibility CRDs are required.

## Stage 0: Preserve the current PoC state

Before implementation:

1. Preserve the current staged, unstaged, and untracked changes in the repository.
2. Record the existing delegated-MaaS implementation for comparison.
3. Avoid overwriting the current partial `MaaSConfiguration` work while introducing the new API.
4. Keep generated files under generator control; do not hand-edit generated CRDs, RBAC, deepcopy code, or webhook manifests.

Exit criteria:

- The implementation starts from a known working-tree baseline.
- Existing user changes remain recoverable and are not silently discarded.

## Stage 1: Define the new API contract

Update the API types in `api/v1alpha1`.

### Core resources

- Use `Organization` as the core organization resource.
- Use `OrganizationProfile` as the API and implementation name.
- Use `OrganizationProject` as the organization-scoped project resource.
- Change references from `spec.tenant` or `spec.tenantRef` to `spec.organizationRef`.
- Keep organization hierarchy and immutable parent references.
- Move display-only names to the `organization.opendatahub.io/display-name` annotation.
- Keep organization policy, project defaults, and shared platform configuration on `OrganizationProfile`.
- Keep MaaS-specific configuration out of `OrganizationProfile`.
- Retain conditions-based status and remove transitional MaaS details from core resource status.

### Shared platform configuration

Add or retain the shared profile configuration for:

- OIDC issuer configuration;
- Kubernetes Gateway API `Gateway` references using group, kind, name, and namespace;
- transitional shared observability settings where required by the design.

The Gateway reference must refer to the shared Kubernetes Gateway API object, not an AI Gateway or MCP Gateway.

### Organization projects

- Rename the project owner reference to `organizationRef`.
- Preserve namespace, RBAC, quota, and network-isolation behavior.
- Add the approved NetworkPolicy grant shape only after confirming the corresponding platform contract.
- Preserve status conditions for namespace and NetworkPolicy readiness.

### MaaS configuration

Create `MaaSConfiguration` as the capability-specific request resource.

- Use `organizationRef`.
- Make the reference immutable.
- Require the resource name to identify the referenced organization, unless the final API contract chooses a different naming rule.
- Keep MaaS quotas and MaaS-specific client settings on this resource.
- Report MaaS readiness through conditions.

## Stage 2: Regenerate API and Kubernetes artifacts

Update Kubebuilder markers and project registration, then regenerate:

- CRDs;
- status and finalizer permissions;
- role and role-binding manifests;
- deepcopy code;
- webhook configurations;
- samples.

Expected API resources are:

- `organizations.organization.opendatahub.io` with short name `org`;
- `organizationprofiles.organization.opendatahub.io` with short name `orgprof`;
- `organizationprojects.organization.opendatahub.io` with short name `orgproj`;
- `maasconfigurations.organization.opendatahub.io` with short name `maascfg`.

The old `tenancy.opendatahub.io` resources should be replaced directly because the API is still a proof of concept.

## Stage 3: Update core controllers

### Organization controller

- Compute and maintain the root organization.
- Create an initial restrictive `OrganizationProfile`.
- Set the profile `organizationRef`.
- Preserve hierarchy validation and owner references.
- Apply organization labels and display-name annotations consistently.

### OrganizationProfile controller

- Keep profile reconciliation minimal and focused on organization policy and shared platform configuration.
- Remove all MaaS-specific reconciliation and status behavior.

### OrganizationProject controller

- Resolve the owning organization through `organizationRef`.
- Provision the namespace, labels, RoleBindings, and status.
- Preserve project quota and authorization rules.
- Manage only tenancy-owned baseline NetworkPolicies.
- Do not delete NetworkPolicies owned by component operators or other controllers.

## Stage 4: Isolate the temporary MaaS implementation

Keep MaaS in this repository, but isolate it from the core organization controllers.

Proposed structure:

```text
api/v1alpha1/maasconfiguration_types.go

internal/controller/maasconfiguration/
  maasconfiguration_controller.go
  maasconfiguration_controller_actions.go
  aitenant_adapter.go
  *_test.go

internal/webhook/v1alpha1/maasconfiguration_webhook.go
```

Isolation requirements:

- Core organization controllers must not import the MaaS controller package.
- MaaS-specific RBAC markers must remain in the MaaS controller package.
- The `AITenant` watch and adapter logic must remain in `aitenant_adapter.go` or a similarly isolated file.
- MaaS registration in `cmd/main.go` must be a clearly removable block.
- MaaS samples and tests must be separately identifiable.
- The MaaS package should depend only on the public organization API, Kubernetes APIs, and its adapter interfaces.

Implement the temporary MaaS behavior with:

- root-organization-only validation;
- immutable `organizationRef`;
- one configuration per root organization;
- self-admin authorization;
- runtime capability-management checks;
- conditions-based readiness and failure reporting;
- explicit `AITenant` owner references and managed-by labels;
- protection against adopting unrelated existing `AITenant` resources;
- finalizer-based cleanup and retry behavior.

The eventual extraction should move the MaaS API/controller, webhook, adapter, tests, RBAC, and samples into the MaaS operator without requiring changes to the core organization controllers.

## Stage 5: Update admission and authorization

Update webhooks and authorization logic for:

- organization parent immutability;
- profile and project `organizationRef` immutability;
- ancestor-admin versus organization self-admin permissions;
- fail-closed handling for missing organization references;
- project quota enforcement;
- valid Gateway API references;
- root-only MaaS configuration;
- duplicate MaaS configuration detection;
- MaaS configuration deletion authorization.

The capability-management check should remain a runtime condition. The MaaS implementation must not mutate the DataScienceCluster component-management setting.

## Stage 6: Expand tests

Update existing tests to use the new API names and fields, then add coverage for:

- organization hierarchy and root calculation;
- automatic restrictive profile creation;
- organization labels and annotations;
- project namespace and RBAC provisioning;
- NetworkPolicy ownership and cleanup;
- short-name discovery and generated CRD shape;
- immutable references and authorization failures;
- root-only and duplicate MaaS configuration validation;
- capability-not-managed status;
- `AITenant` rendering, ownership, readiness, and foreign-resource protection;
- finalizer cleanup, retry, and partial-failure behavior.

## Stage 7: Update documentation and demos

Update `README.md`, sample manifests, demo scripts, and RBAC examples to use:

```sh
kubectl get org,orgprof,orgproj,maascfg
```

Documentation must use the Organization terminology, `organizationRef`, organization labels, the shared Kubernetes Gateway API terminology, and the temporary MaaS extraction boundary.

## Stage 8: Verification

Run the repository checks in this order:

```sh
make manifests
make generate
make fmt
make vet
make test
make lint-fix
```

Then run the isolated kind/e2e tests and verify API discovery:

```sh
kubectl api-resources | grep -E 'organizations|organizationprofiles|organizationprojects|maasconfigurations'
```

Final acceptance criteria:

- Core resources use the Organization API names and group.
- All four short names are available through API discovery.
- Organization controllers have no dependency on MaaS implementation packages.
- MaaS behavior is isolated and movable to another operator.
- Generated artifacts are reproducible.
- Existing unmanaged MaaS resources are not adopted or deleted.
- Tests cover authorization, ownership, readiness, deletion, and retry behavior.
