package gha

var (
	archSuffixes = []string{
		"-linux-amd64",
		"-linux-arm64",
		"-windows-amd64",
		"-windows-arm64",
		"-windows-ltsc2022-amd64",
		"-windows-ltsc2022-arm64",
		"-amd64",
		"-arm64",
		"-s390x",
	}

	// imageRepo holds the mappings between container image and source code repositories.
	imageRepo = map[string]string{
		"rancher/rancher-csp-adapter":                         "rancher/csp-adapter",
		"rancher/fleet-agent":                                 "rancher/fleet",
		"rancher/rke2-runtime":                                "rancher/rke2",
		"rancher/rke2-cloud-provider":                         "rancher/image-build-rke2-cloud-provider",
		"rancher/hardened-addon-resizer":                      "rancher/image-build-addon-resizer",
		"rancher/hardened-calico":                             "rancher/image-build-calico",
		"rancher/hardened-cluster-autoscaler":                 "rancher/image-build-cluster-proportional-autoscaler",
		"rancher/hardened-whereabouts":                        "rancher/image-build-whereabouts",
		"rancher/hardened-node-feature-discovery":             "rancher/image-build-node-feature-discovery",
		"rancher/hardened-multus-cni":                         "rancher/image-build-multus",
		"rancher/hardened-multus-thick":                       "rancher/image-build-multus",
		"rancher/hardened-multus-dynamic-networks-controller": "rancher/image-build-multus-dynamic-networks-controller",
		"rancher/hardened-kubernetes":                         "rancher/image-build-kubernetes",
		"rancher/hardened-k8s-metrics-server":                 "rancher/image-build-k8s-metrics-server",
		"rancher/hardened-flannel":                            "rancher/image-build-flannel",
		"rancher/hardened-etcd":                               "rancher/image-build-etcd",
		"rancher/hardened-dns-node-cache":                     "rancher/image-build-dns-nodecache",
		"rancher/hardened-coredns":                            "rancher/image-build-coredns",
		"rancher/hardened-cni-plugins":                        "rancher/image-build-cni-plugins",
		"rancher/nginx-ingress-controller":                    "rancher/ingress-nginx",
		"rancher/nginx-ingress-controller-chroot":             "rancher/ingress-nginx",
		"rancher/rancher":                                     "rancher/rancher-prime",
		"rancher/neuvector-manager":                           "neuvector/manager",
		"rancher/neuvector-controller":                        "neuvector/neuvector",
		"rancher/neuvector-enforcer":                          "neuvector/neuvector",
		"rancher/neuvector-scanner":                           "neuvector/scanner",
		"rancher/neuvector-prometheus-exporter":               "neuvector/prometheus-exporter",
		"rancher/neuvector-registry-adapter":                  "neuvector/registry-adapter",
		"rancher/neuvector-updater":                           "neuvector/updater",
		"rancher/neuvector-compliance-config":                 "neuvector/compliance-config",
		"rancher/supportability-review-internal":              "rancher/supportability-review",
		"rancher/supportability-review-app-frontend":          "rancher/supportability-review-operator",
	}

	mutableRepo = map[string]bool{
		"rancher/neuvector-scanner:6": true,
	}

	identityOverride = map[string]string{
		"rancher/cluster-api-addon-provider-fleet":                        "^https://github.com/rancher/clusterapi-forks/.github/workflows/caapf.yaml@refs/heads/main$",
		"rancher/ip-address-manager":                                      "^https://github.com/rancher/clusterapi-forks/.github/workflows/metal3-ipam.yaml@refs/heads/main$",
		"rancher/cluster-api-controller":                                  "^https://github.com/rancher/clusterapi-forks/.github/workflows/core.yaml@refs/heads/main$",
		"rancher/cluster-api-aws-controller":                              "^https://github.com/rancher/clusterapi-forks/.github/workflows/aws.yaml@refs/heads/main$",
		"rancher/cluster-api-azure-controller":                            "^https://github.com/rancher/clusterapi-forks/.github/workflows/azure.yaml@refs/heads/main$",
		"rancher/cluster-api-gcp-controller":                              "^https://github.com/rancher/clusterapi-forks/.github/workflows/gcp.yaml@refs/heads/main$",
		"rancher/cluster-api-vsphere-controller":                          "^https://github.com/rancher/clusterapi-forks/.github/workflows/vsphere.yaml@refs/heads/main$",
		"rancher/cluster-api-metal3-controller":                           "^https://github.com/rancher/clusterapi-forks/.github/workflows/metal3.yaml@refs/heads/main$",
		"rancher/cluster-api-metal3-ipam-controller":                      "^https://github.com/rancher/clusterapi-forks/.github/workflows/metal3-ipam.yaml@refs/heads/main$",
		"rancher/mirrored-cilium-cilium":                                  "^https://github.com/cilium/cilium/.github/workflows/build-images-releases.yaml@refs/tags/v",
		"rancher/mirrored-cilium-envoy":                                   "^https://github.com/cilium/proxy/.github/workflows/build-envoy-images-release.yaml@refs/heads/v",
		"rancher/mirrored-cilium-clustermesh-apiserver":                   "^https://github.com/cilium/cilium/.github/workflows/build-images-releases.yaml@refs/tags/v",
		"rancher/mirrored-cilium-hubble-relay":                            "^https://github.com/cilium/cilium/.github/workflows/build-images-releases.yaml@refs/tags/v",
		"rancher/mirrored-cilium-operator-aws":                            "^https://github.com/cilium/cilium/.github/workflows/build-images-releases.yaml@refs/tags/v",
		"rancher/mirrored-cilium-operator-azure":                          "^https://github.com/cilium/cilium/.github/workflows/build-images-releases.yaml@refs/tags/v",
		"rancher/mirrored-cilium-operator-generic":                        "^https://github.com/cilium/cilium/.github/workflows/build-images-releases.yaml@refs/tags/v",
		"rancher/mirrored-prometheus-operator-prometheus-config-reloader": "^https://github.com/prometheus-operator/prometheus-operator/.github/workflows/publish.yaml@refs/tags/v",
		"rancher/mirrored-kube-logging-logging-operator":                  "^https://github.com/kube-logging/logging-operator/.github/workflows/artifacts.yaml@refs/tags/",
		"rancher/image-build-etcd":                                        "^https://github.com/rancher/image-build-etcd/.github/workflows/(image-push|release).yml@refs/tags/v",
		"rancher/image-build-whereabouts":                                 "^https://github.com/rancher/image-build-whereabouts/.github/workflows/(image-push|release).yml@refs/tags/v",
		"rancher/image-build-rke2-cloud-provider":                         "^https://github.com/rancher/image-build-rke2-cloud-provider/.github/workflows/(image-push|release).yml@refs/tags/v",
		"rancher/image-build-cni-plugins":                                 "^https://github.com/rancher/image-build-cni-plugins/.github/workflows/(image-push|release).yml@refs/tags/v",
		"rancher/supportability-review":                                   "^https://github.com/rancher/supportability-review/.github/workflows/release.yaml@refs/tags/v",
		"rancher/rancher-prime":                                           "^https://github.com/rancher/rancher-prime/.github/workflows/(release|alpha-release|rc-release).yml@refs/tags/v",
		"rancher/prometheus-federator":                                    "^https://github.com/rancher/prometheus-federator/.github/workflows/(release|publish).yaml@refs/tags/v",
	}

	// imageSuffixes holds a mapping between image name and the ref suffixes
	// they may have which will need to be trimmed before defining the expected
	// subject identity.
	imageSuffixes = map[string][]string{
		"rancher/hardened-multus-cni": {"-arch"},
	}
)
