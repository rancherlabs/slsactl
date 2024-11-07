# slsactl

The `slsactl` is a Command Line Interface (CLI) tool to provide a consolidated
experience while handling supply chain aspects of projects across the Rancher
ecosystem. 

## Installation

### Go

```bash
go install github.com/rancherlabs/slsactl@latest
```

## Usage

### Provenance
The provenance data can be extracted from an image with the command below.
Note that the image must have been built with a Provenance layer baked to it.

```bash
slsactl download provenance rancher/cis-operator:v1.0.15
slsactl download provenance --format=slsav1 rancher/cis-operator:v1.0.15
```

By default, the returned provenance would be for `linux/amd64`. To select a
different platform use `--platform`.

### SBOM
The latest container images have baked into them a layer containing their SPDX
SBOM, which can be extracted with:

```bash
slsactl download sbom rancher/cis-operator:v1.0.15
```

If Cyclonedx is required instead:

```bash
slsactl download sbom -format cyclonedxjson rancher/cis-operator:v1.0.15
```

By default, the returned provenance would be for `linux/amd64`. To select a
different platform use `--platform`.

Note that images that haven't got a SBOM layer attached to them, the same
command will generate a SBOM manifest on-demand, which will take longer.
An example being:

```bash
slsactl download sbom rancher/rancher:v2.8.1
```

### Verify
The cosign verification of Rancher Prime images can be done with:

```bash
slsactl verify <prime_image>:<tag>
```

## License
Copyright (c) 2014-2024 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](LICENSE)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
