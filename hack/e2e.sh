#!/bin/bash

set -eox pipefail

IMAGE=${IMAGE:-ghcr.io/kubewarden/policy-server:v1.19.0}

slsactl verify "${IMAGE}"
slsactl download provenance "${IMAGE}"
slsactl download provenance --format=slsav1 "${IMAGE}"
slsactl download sbom "${IMAGE}"
slsactl download sbom -format cyclonedxjson "${IMAGE}"
slsactl version
