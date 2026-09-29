#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
CONTAINER_TOOL="${CONTAINER_TOOL:-podman}"
KUBECTL="${KUBECTL:-oc}"
IMAGE_REPOSITORY="${IMAGE_REPOSITORY:-quay.io/lphiri/tenancy-operator}"
IMAGE_TAG="${IMAGE_TAG:-poc-$(date +%Y%m%d%H%M%S)}"
IMAGE="${IMAGE:-${IMAGE_REPOSITORY}:${IMAGE_TAG}}"
NAMESPACE="${NAMESPACE:-tenancy-operator-system}"
DEPLOYMENT="${DEPLOYMENT:-tenancy-operator-controller-manager}"
ROLLOUT_TIMEOUT="${ROLLOUT_TIMEOUT:-120s}"

usage() {
	cat <<EOF
Build, push, and deploy the tenancy operator from Quay.

Usage: $(basename "$0")

Environment overrides:
  CONTAINER_TOOL    Container tool to use (default: podman)
  KUBECTL           Kubernetes CLI to use (default: oc)
  IMAGE_REPOSITORY  Quay image repository
  IMAGE_TAG         Image tag (default: poc-<timestamp>)
  IMAGE             Complete image reference; overrides repository and tag
  NAMESPACE         Operator namespace (default: tenancy-operator-system)
  DEPLOYMENT        Controller deployment name
  ROLLOUT_TIMEOUT   Rollout wait timeout (default: 120s)

The container tool must already be logged in to quay.io, and KUBECTL must
already be logged in to the target cluster.
EOF
}

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "Required command not found: $1" >&2
		exit 1
	fi
}

case "${1:-}" in
"" ) ;;
--help|-h)
	usage
	exit 0
	;;
*)
	echo "Unknown option: $1" >&2
	usage >&2
	exit 2
	;;
esac

require_command make
require_command "$CONTAINER_TOOL"
require_command "$KUBECTL"

cd "$REPO_ROOT"

echo "Using Kubernetes context: $($KUBECTL config current-context)"
echo "Building and pushing: $IMAGE"

CONTAINER_TOOL="$CONTAINER_TOOL" make docker-build IMG="$IMAGE"
CONTAINER_TOOL="$CONTAINER_TOOL" make docker-push IMG="$IMAGE"
CONTAINER_TOOL="$CONTAINER_TOOL" KUBECTL="$KUBECTL" make deploy IMG="$IMAGE"

"$KUBECTL" -n "$NAMESPACE" rollout status \
	"deployment/$DEPLOYMENT" --timeout="$ROLLOUT_TIMEOUT"

echo "Deployed $IMAGE"
