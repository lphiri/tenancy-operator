# OpenShift demo from Quay

This guide runs the tenancy demo on an OpenShift cluster using the published
controller image:

```text
quay.io/lphiri/tenancy-operator:poc
```

The existing [kind demo](README.md#demo) remains available for local
development. This guide assumes that you already have an OpenShift cluster and
are using a cluster-admin context for installation and the impersonation-based
authorization demo.

## Prerequisites

- `oc`, `kubectl`, `git`, and `make`
- An OpenShift cluster and a cluster-admin login
- cert-manager installed and healthy
- Access to `quay.io/lphiri/tenancy-operator:poc`

The operator uses cert-manager for its webhook certificates. If cert-manager
is not already installed, install it using the OpenShift cert-manager Operator
or the [official static installation](https://cert-manager.io/docs/installation/kubectl/).
For a quick proof of concept with the current supported release:

```sh
oc apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.21.2/cert-manager.yaml
oc -n cert-manager wait --for=condition=Available deployment/cert-manager --timeout=120s
oc -n cert-manager wait --for=condition=Available deployment/cert-manager-cainjector --timeout=120s
oc -n cert-manager wait --for=condition=Available deployment/cert-manager-webhook --timeout=120s
```

## Deploy the controller from Quay

Log in to the OpenShift cluster and select the context you want to use:

```sh
oc login https://api.<cluster-domain>:6443
export CTX="$(oc config current-context)"
export IMAGE=quay.io/lphiri/tenancy-operator:poc
```

Deploy the CRDs, RBAC, webhook, and controller. `KUBECTL=oc` makes the
repository's deployment target use the OpenShift CLI:

```sh
CONTAINER_TOOL=podman KUBECTL=oc make deploy IMG="$IMAGE"
oc -n tenancy-operator-system rollout status \
  deployment/tenancy-operator-controller-manager --timeout=120s
```

If the Quay repository is private, create a pull secret and attach it to the
controller service account:

```sh
export QUAY_USERNAME=<quay-username>
export QUAY_TOKEN=<quay-token>

oc -n tenancy-operator-system create secret docker-registry quay-pull-secret \
  --docker-server=quay.io \
  --docker-username="$QUAY_USERNAME" \
  --docker-password="$QUAY_TOKEN"
oc -n tenancy-operator-system secrets link \
  tenancy-operator-controller-manager quay-pull-secret --for=pull
oc -n tenancy-operator-system rollout restart \
  deployment/tenancy-operator-controller-manager
oc -n tenancy-operator-system rollout status \
  deployment/tenancy-operator-controller-manager --timeout=120s
```

Verify that the CRDs and controller are available:

```sh
oc get crd organizations.organization.opendatahub.io
oc get crd organizationprofiles.organization.opendatahub.io
oc get crd organizationprojects.organization.opendatahub.io
oc get crd maasconfigurations.organization.opendatahub.io
oc -n tenancy-operator-system get pods
```

## Run the demo

The demo uses Kubernetes impersonation to exercise the validating webhooks, so
run it with a context authorized to impersonate `alice`, `bob`, and `carol`:

```sh
oc apply -f hack/demo-rbac.yaml
CTX="$CTX" ./hack/demo.sh
```

The demo creates inspectable YAML resources and covers:

- delegated organization administration;
- organization projects, namespaces, RBAC, and NetworkPolicies;
- a grandchild organization with its own administrator;
- the maximum hierarchy depth of three;
- shared OIDC issuer and ingress Gateway settings;
- root-only `MaaSConfiguration` authorization.

The OpenShift cluster does not need the external MaaS `AITenant` CRD for this
demo. The root `MaaSConfiguration` is accepted and reports that the capability
is unavailable; the child configuration is rejected because MaaS is root-only.

## Cleanup

Remove the demo resources and the operator installation with:

```sh
oc delete maascfg research-division genomics --ignore-not-found
oc delete orgproj genomics-analysis genomics-analysis-2 genomics-analysis-3 proteomics-analysis --ignore-not-found
oc delete org research-division genomics proteomics depth-04 --ignore-not-found
oc delete -f hack/demo-rbac.yaml --ignore-not-found
CONTAINER_TOOL=podman KUBECTL=oc make undeploy ignore-not-found=true
```

Delete MaaS configurations before their organizations so the validating
webhook can complete finalizer cleanup. The final `make undeploy` removes the
operator deployment, webhook, RBAC, and CRDs.
