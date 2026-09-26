# tenancy-operator

A proof-of-concept Kubernetes operator for the RHOAI organization and
capability framework. It implements hierarchical organizations, delegated
administration, organization projects, and a temporarily co-located MaaS
configuration controller.

## API

The cluster-scoped API is `organization.opendatahub.io/v1alpha1`:

- **Organization** (`org`) is a node in the organization tree. `spec.parent`
  links to its parent; an empty parent creates a root organization. The
  reconciler computes `status.root` and creates an initial restrictive
  **OrganizationProfile**.
- **OrganizationProfile** (`orgprof`) contains organization administrators,
  project defaults, and shared platform configuration. Its
  `spec.organizationRef` is immutable and its name must match the referenced
  organization.
- **OrganizationProject** (`orgproj`) provisions one workload namespace with
  organization labels, edit/view RoleBindings, and tenancy-owned baseline
  NetworkPolicies. It references its organization through
  `spec.organizationRef`.
- **MaaSConfiguration** (`maascfg`) requests MaaS for one root organization.
  The temporary MaaS implementation renders an owned `AITenant` when that CRD
  is installed. Its controller and webhook are isolated so they can later move
  to the MaaS operator.

The short names allow a complete view with:

```sh
kubectl get org,orgprof,orgproj,maascfg
```

Organization labels use the `organization.opendatahub.io/*` prefix. Display
names are annotations rather than placement or authorization fields.

## Authorization

The validating webhooks fail closed:

- Organization creation and hierarchy changes require ancestor-admin authority;
- profile and project administration use the organization hierarchy and
  self-admin rules defined by the API contract;
- `spec.parent` and each `spec.organizationRef` are immutable;
- MaaS configuration is root-only, has one configuration per root organization,
  and requires self-admin authorization;
- cluster-admin groups and the operator ServiceAccount are trusted;
- cluster-admin groups default to `system:masters` and
  `kubeadm:cluster-admins`, and can be overridden with
  `CLUSTER_ADMIN_GROUPS`.

MaaS capability availability is reported at runtime. The MaaS controller does
not modify the DataScienceCluster component-management setting.

## Getting Started

Prerequisites: Go 1.24+, kind, kubectl, podman or docker, and cert-manager.

```sh
kind create cluster --name tenancy-poc
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.21.2/cert-manager.yaml
kubectl -n cert-manager rollout status deploy/cert-manager-webhook

CONTAINER_TOOL=podman make docker-build IMG=localhost/tenancy-operator:poc
podman save localhost/tenancy-operator:poc > /tmp/op.tar
kind load image-archive /tmp/op.tar --name tenancy-poc
make deploy IMG=localhost/tenancy-operator:poc
```

For an OpenShift cluster using the published Quay image, see
[QUAY_DEMO.md](QUAY_DEMO.md).

## Demo

`hack/demo.sh` demonstrates organization delegation, ancestor versus self
authority, parent immutability, grandchild administration, project provisioning,
project quotas, shared OIDC and ingress gateway settings, the organization
hierarchy depth limit, and root-only MaaSConfiguration authorization.
The scenario resources are separate, inspectable YAML files under `hack/demo/`.
The hierarchy controller limits organization trees to three levels; the demo
shows the third level beside a failed fourth-level organization.
The MaaS portion intentionally shows the capability-unavailable status when
the external MaaS `AITenant` CRD is not installed.

```sh
kubectl apply -f hack/demo-rbac.yaml
hack/demo.sh
```

## License

Copyright 2026. Licensed under the Apache License, Version 2.0.
