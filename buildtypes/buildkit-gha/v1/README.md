# Build Type: BuildKit + GitHub Actions

This is a [SLSA Provenance](https://slsa.dev/provenance/v1)
`buildType` that describes builds for container images which combine the
use of BuildKit v1 and GitHub Actions workflows.

## Description

```jsonc
"buildType": "https://github.com/rancherlabs/slsactl/tree/main/buildtypes/buildkit-gha/v1"
```

## Build Definition

### Internal parameters

All internal parameters are REQUIRED.

| Parameter | Type | Description |
| --------- | ---- | ----------- |
| `trigger` | string | The GitHub Action event that caused the build to be executed. |
| `invocationUri` | string | Resource URI for the GitHub action workflow instance. |

### Resolved Dependencies

The resolved dependencies MUST include the source code URI and its `gitCommit`, optionally
a Git tag can be added as an annotation.
