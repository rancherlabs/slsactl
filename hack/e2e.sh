#!/bin/bash

set -eox pipefail

SLSACTL=./build/bin/slsactl

# First image uses old signature format, the second the new.
IMAGES=("ghcr.io/kubewarden/policy-server:v1.19.0" "ghcr.io/kubewarden/policy-server:v1.31.0")
IMAGE="${IMAGE:-}"

for IMAGE in "${IMAGES[@]}"; do
    ${SLSACTL} verify "${IMAGE}"
    ${SLSACTL} download provenance "${IMAGE}"
    ${SLSACTL} download provenance --format=slsav1 "${IMAGE}"
    ${SLSACTL} download sbom "${IMAGE}"
    ${SLSACTL} download sbom -format cyclonedxjson "${IMAGE}"
    ${SLSACTL} version
done
