package verify

// imageRepo holds the mappings between container image and source code repositories.
var imageRepo = map[string]string{
	"rancher/rke2-runtime":                    "rancher/rke2",
	"rancher/rke2-cloud-provider":             "rancher/image-build-rke2-cloud-provider",
	"rancher/hardened-addon-resizer":          "rancher/image-build-addon-resizer",
	"rancher/hardened-calico":                 "rancher/image-build-calico",
	"rancher/hardened-cluster-autoscaler":     "rancher/image-build-cluster-proportional-autoscaler",
	"rancher/hardened-whereabouts":            "rancher/image-build-whereabouts",
	"rancher/hardened-node-feature-discovery": "rancher/image-build-node-feature-discovery",
	"rancher/hardened-multus-cni":             "rancher/image-build-multus",
	"rancher/hardened-kubernetes":             "rancher/image-build-kubernetes",
	"rancher/hardened-k8s-metrics-server":     "rancher/image-build-k8s-metrics-server",
	"rancher/hardened-flannel":                "rancher/image-build-flannel",
	"rancher/hardened-etcd":                   "rancher/image-build-etcd",
	"rancher/hardened-dns-node-cache":         "rancher/image-build-dns-nodecache",
	"rancher/hardened-coredns":                "rancher/image-build-coredns",
	"rancher/hardened-cni-plugins":            "rancher/image-build-cni-plugins",
	"rancher/nginx-ingress-controller":        "rancher/ingress-nginx",
}
