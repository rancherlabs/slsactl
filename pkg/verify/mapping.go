package verify

import (
	"fmt"
	"strings"
)

type identityIssuer struct {
	identity string
	issuer   string
}

var nonGitHub = map[string]identityIssuer{
	"sig-storage/snapshot-controller": {
		identity: "krel-trust@k8s-releng-prod.iam.gserviceaccount.com",
		issuer:   "https://accounts.google.com",
	},
	"sig-storage/snapshot-validation-webhook": {
		identity: "krel-trust@k8s-releng-prod.iam.gserviceaccount.com",
		issuer:   "https://accounts.google.com",
	},
	"rancher/mirrored-sig-storage-csi-node-driver-registrar": {
		identity: "krel-trust@k8s-releng-prod.iam.gserviceaccount.com",
		issuer:   "https://accounts.google.com",
	},
	"rancher/mirrored-sig-storage-csi-attacher": {
		identity: "krel-trust@k8s-releng-prod.iam.gserviceaccount.com",
		issuer:   "https://accounts.google.com",
	},
	"rancher/mirrored-sig-storage-csi-provisioner": {
		identity: "krel-trust@k8s-releng-prod.iam.gserviceaccount.com",
		issuer:   "https://accounts.google.com",
	},
	"rancher/mirrored-sig-storage-csi-resizer": {
		identity: "krel-trust@k8s-releng-prod.iam.gserviceaccount.com",
		issuer:   "https://accounts.google.com",
	},
	"rancher/mirrored-sig-storage-csi-snapshotter": {
		identity: "krel-trust@k8s-releng-prod.iam.gserviceaccount.com",
		issuer:   "https://accounts.google.com",
	},
	"rancher/mirrored-sig-storage-livenessprobe": {
		identity: "krel-trust@k8s-releng-prod.iam.gserviceaccount.com",
		issuer:   "https://accounts.google.com",
	},
	"rancher/mirrored-sig-storage-snapshot-controller": {
		identity: "krel-trust@k8s-releng-prod.iam.gserviceaccount.com",
		issuer:   "https://accounts.google.com",
	},
}

// imageRepo holds the mappings between container image and source code repositories.
var imageRepo = map[string]string{
	"rancher/rancher-csp-adapter":             "rancher/csp-adapter",
	"rancher/fleet-agent":                     "rancher/fleet",
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
	"rancher/rancher":                         "rancher/rancher-prime",
	"rancher/neuvector-manager":               "neuvector/manager",
	"rancher/neuvector-controller":            "neuvector/neuvector",
	"rancher/neuvector-enforcer":              "neuvector/neuvector",
	"rancher/neuvector-scanner":               "neuvector/scanner",
	"rancher/neuvector-prometheus-exporter":   "neuvector/prometheus-exporter",
	"rancher/neuvector-registry-adapter":      "neuvector/registry-adapter",
	"rancher/neuvector-updater":               "neuvector/updater",
	"rancher/neuvector-compliance-config":     "neuvector/compliance-config",
}

var mutableRepo = map[string]bool{
	"rancher/neuvector-scanner:6": true,
}

var obs = map[string]struct{}{
	"rancher/elemental-operator":            {},
	"rancher/seedimage-builder":             {},
	"rancher/elemental-channel/sl-micro":    {},
	"rancher/elemental-operator-crds-chart": {},
	"rancher/elemental-operator-chart":      {},
	"bci/bci-base":                          {},
	"bci/bci-busybox":                       {},
	"suse/sles/15.7/cdi-cloner":             {},
	"suse/sles/15.7/cdi-controller":         {},
	"suse/sles/15.7/cdi-importer":           {},
	"suse/sles/15.7/cdi-operator":           {},
	"suse/sles/15.7/cdi-uploadproxy":        {},
	"suse/sles/15.7/cdi-uploadserver":       {},
	"suse/sles/15.7/libguestfs-tools":       {},
	"suse/sles/15.7/virt-api":               {},
	"suse/sles/15.7/virt-controller":        {},
	"suse/sles/15.7/virt-handler":           {},
	"suse/sles/15.7/virt-launcher":          {},
	"suse/sles/15.7/virt-operator":          {},
	"suse/vmdp/vmdp":                        {},
}

func obsSigned(image string) bool {
	bef, after, _ := strings.Cut(image, "/")
	if strings.Contains(bef, ".") {
		image = after
	}

	bef, _, _ = strings.Cut(image, ":")
	image = bef
	fmt.Println(image)

	_, ok := obs[image]
	return ok
}

var upstreamImageRepo = map[string]string{
	"rancher/cluster-api-controller":             "https://github.com/rancher/clusterapi-forks/.github/workflows/core.yaml@refs/heads/main",
	"rancher/cluster-api-aws-controller":         "https://github.com/rancher/clusterapi-forks/.github/workflows/aws.yaml@refs/heads/main",
	"rancher/cluster-api-azure-controller":       "https://github.com/rancher/clusterapi-forks/.github/workflows/azure.yaml@refs/heads/main",
	"rancher/cluster-api-gcp-controller":         "https://github.com/rancher/clusterapi-forks/.github/workflows/gcp.yaml@refs/heads/main",
	"rancher/cluster-api-vsphere-controller":     "https://github.com/rancher/clusterapi-forks/.github/workflows/vsphere.yaml@refs/heads/main",
	"rancher/cluster-api-metal3-controller":      "https://github.com/rancher/clusterapi-forks/.github/workflows/metal3.yaml@refs/heads/main",
	"rancher/cluster-api-metal3-ipam-controller": "https://github.com/rancher/clusterapi-forks/.github/workflows/metal3-ipam.yaml@refs/heads/main",
}

// imageSuffixes holds a mapping between image name and the ref suffixes
// they may have which will need to be trimmed before defining the expected
// subject identity.
var imageSuffixes = map[string][]string{
	"rancher/hardened-multus-cni": {"-arch"},
}
