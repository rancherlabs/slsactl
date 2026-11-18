#!/bin/bash

set -eox pipefail

SLSACTL=./build/bin/slsactl

# The kubewarden/policy-server images cover the old and the new signature
# formats. The adm-controller images cover the per-repository release
# workflow identity (v1.38.2) and the shared container-release.yml
# workflow identity (v1.39.0).
IMAGES=(
    "ghcr.io/kubewarden/policy-server:v1.19.0"
    "ghcr.io/kubewarden/policy-server:v1.31.0"
    "ghcr.io/kubewarden/adm-controller/policy-server:v1.38.2"
    "ghcr.io/kubewarden/adm-controller/policy-server:v1.39.0"
)
IMAGE="${IMAGE:-}"

for IMAGE in "${IMAGES[@]}"; do
    ${SLSACTL} verify "${IMAGE}"
    ${SLSACTL} download provenance "${IMAGE}"
    ${SLSACTL} download provenance --format=slsav1 "${IMAGE}"
    ${SLSACTL} download sbom "${IMAGE}"
    ${SLSACTL} download sbom -format cyclonedxjson "${IMAGE}"
    ${SLSACTL} version
done
