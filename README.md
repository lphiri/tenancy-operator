# tenancy-operator

A proof-of-concept Kubernetes operator for the RHOAI multi-tenancy framework
([ODH-ADR-Operator-0015](../architecture-decision-records/architecture-decision-records/operator/ODH-ADR-Operator-0015-multi-tenancy-framework.md)).
It implements a hierarchy of tenants, top-down delegated administration, and a
fail-closed validating webhook that is the authorization gate for all tenant
changes.

## Model

Three cluster-scoped CRDs in group `tenancy.opendatahub.io/v1alpha1`:

- **PlatformTenant** - a node in the tenant tree. `spec.parent` links to its
  parent (empty = root). The reconciler computes `status.root` by walking the
  chain and auto-creates a restrictive **TenantProfile** (maxProjects=0,
  networkIsolation=tenant) for every new tenant.
- **TenantProfile** - per-tenant config (name must equal `spec.tenant`):
  `spec.admins` and `spec.defaults` (networkIsolation none/tenant/strict,
  maxProjects). Self-managed: the operator never overwrites an existing one.
- **TenantProject** - a workload namespace under a tenant. The reconciler
  provisions the Namespace (labeled with tenant + root), edit/view RoleBindings
  from `spec.users`, and NetworkPolicies matching the effective isolation.

### Authorization (validating webhook, failurePolicy=Fail)

- **Existence, placement, and admin assignment** require *ancestor-admin*
  authority: the requester is an admin on the tenant or any ancestor.
- **Config self-management** (profile defaults) requires *self-admin*: an admin
  listed on the tenant's own profile. Ancestors do not inherit config rights.
- `spec.parent` (PlatformTenant) and `spec.tenant` (TenantProfile/TenantProject)
  are immutable.
- Cluster-admins and the operator's own ServiceAccount are trusted and bypass
  these checks. Cluster-admin groups default to `system:masters` and
  `kubeadm:cluster-admins`; override with the `CLUSTER_ADMIN_GROUPS` env var.

## Getting Started

Prereqs: Go 1.24+, kind (podman or docker), kubectl, cert-manager.

```sh
# 1. Create a cluster and install cert-manager (webhook TLS).
kind create cluster --name tenancy-poc
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.2/cert-manager.yaml
kubectl -n cert-manager rollout status deploy/cert-manager-webhook

# 2. Build, load, and deploy the operator.
make docker-build IMG=localhost/tenancy-operator:poc
podman save localhost/tenancy-operator:poc -o /tmp/op.tar && kind load image-archive /tmp/op.tar --name tenancy-poc
make deploy IMG=localhost/tenancy-operator:poc
```

## Demo

`hack/demo.sh` walks the full authorization model using kubectl impersonation
(cluster-admin bootstraps a root tenant; `alice` is a delegated admin; `bob` has
no authority). It demonstrates delegation, ancestor vs. self authority, parent
immutability, project provisioning, and maxProjects quota enforcement.

```sh
kubectl apply -f hack/demo-rbac.yaml   # let demo users reach the CRDs; the webhook is the gate
hack/demo.sh
```

## License

Copyright 2026. Licensed under the Apache License, Version 2.0.
