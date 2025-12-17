#!/bin/bash

set -eox pipefail

# First image uses old signature format, the second the new.
IMAGES=("ghcr.io/kubewarden/policy-server:v1.19.0" "ghcr.io/kubewarden/policy-server:v1.31.0")
IMAGE="${IMAGE:-}"

for IMAGE in "${IMAGES[@]}"; do
    slsactl verify "${IMAGE}"
    slsactl download provenance "${IMAGE}"
    slsactl download provenance --format=slsav1 "${IMAGE}"
    slsactl download sbom "${IMAGE}"
    slsactl download sbom -format cyclonedxjson "${IMAGE}"
    slsactl version
done
