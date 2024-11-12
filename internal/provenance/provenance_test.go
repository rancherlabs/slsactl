package provenance_test

import (
	"encoding/json"
	"testing"
	"time"

	_ "embed"

	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	v02 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	v1 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	"github.com/rancherlabs/slsactl/internal/provenance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed security-scan.v0.2
var v02Data []byte

func TestConvertV02ToV1(t *testing.T) {
	var v02Prov v02.ProvenancePredicate
	err := json.Unmarshal(v02Data, &v02Prov)
	require.NoError(t, err, "Failed to unmarshal v0.2 data")

	v1Prov := provenance.ConvertV02ToV1(v02Prov)

	started, _ := time.Parse(time.RFC3339Nano, "2024-07-11T14:49:18.126688014Z")
	finished, _ := time.Parse(time.RFC3339Nano, "2024-07-11T14:51:00.499751748Z")

	expectedV1Prov := v1.ProvenancePredicate{
		BuildDefinition: v1.ProvenanceBuildDefinition{
			BuildType: "https://mobyproject.org/buildkit@v1",
			ExternalParameters: map[string]interface{}{
				"args": map[string]string{
					"build-arg:KUBECTL_SUM_amd64":    "aff42d3167685e4d8e86fda0ad9c6ce6ec6c047bc24d608041d54717a18192ba",
					"build-arg:KUBECTL_SUM_arm64":    "13d547495bdea49b223fe06bffb6d2bef96436634847f759107655aa80fc990e",
					"build-arg:KUBECTL_VERSION":      "1.28.7",
					"build-arg:KUBE_BENCH_SUM_amd64": "8e8f083819678956b6c36623a6a0638741340397ffc209cd71a6b4907f2bb05e",
					"build-arg:KUBE_BENCH_SUM_arm64": "82256042da9d78bb1cf1726dc8c108459c3cdc34df6298349113f551bde0feff",
					"build-arg:KUBE_BENCH_VERSION":   "v0.8.0",
					"build-arg:SONOBUOY_SUM_amd64":   "0fd3ae735ee25b6b37b713aadd4a836b53aa2b82c8e6ecad0c2359de046f8212",
					"build-arg:SONOBUOY_SUM_arm64":   "b6665011ae337e51cd6032ebfc37a6818dc8481c6cf6f2692e109a08be908e49",
					"build-arg:SONOBUOY_VERSION":     "v0.57.1",
					"build-arg:VERSION":              "v0.0.6",
				},
				"frontend": "dockerfile.v0",
				"locals": []struct{ name string }{
					{name: "context"},
					{name: "dockerfile"},
				},
			},
			InternalParameters: map[string]interface{}{
				"platform": "linux/amd64",
			},
			ResolvedDependencies: []v1.ResourceDescriptor{
				{
					URI: "pkg:docker/docker/buildkit-syft-scanner@stable-1",
					Digest: common.DigestSet{
						"sha256": "176e0869c38aeaede37e594fcf182c91d44391a932e1d71e99ec204873445a33",
					},
				},
				{
					URI: "pkg:docker/rancher/mirrored-tonistiigi-xx@1.3.0?platform=linux%2Famd64",
					Digest: common.DigestSet{
						"sha256": "053f8e16c843695b7a23803fbfdd699a8b9c9fe863a613516e4911a6eba0a4cb",
					},
				},
				{
					URI: "pkg:docker/registry.suse.com/bci/bci-micro@15.6?platform=linux%2Famd64",
					Digest: common.DigestSet{
						"sha256": "8f926e98dd809e5fc5971e39df2d88a7dbe4158dcf2c379be658acc67b1beb29",
					},
				},
				{
					URI: "pkg:docker/registry.suse.com/bci/golang@1.22?platform=linux%2Famd64",
					Digest: common.DigestSet{
						"sha256": "fdf2b123574c9b00ee19a7009c6b8b11c4e97f3dcb5e27c7ab49d23f8e722d21",
					},
				},
				{
					URI: "https://dl.k8s.io/release/v1.28.7/bin/linux/amd64/kubectl",
					Digest: common.DigestSet{
						"sha256": "aff42d3167685e4d8e86fda0ad9c6ce6ec6c047bc24d608041d54717a18192ba",
					},
				},
			},
		},
		RunDetails: v1.ProvenanceRunDetails{
			Builder: v1.Builder{
				ID: "",
			},
			BuildMetadata: v1.BuildMetadata{
				StartedOn:    &started,
				FinishedOn:   &finished,
				InvocationID: "ujss3xdtnmbv38uqh5wwftkpd",
			},
			Byproducts: []v1.ResourceDescriptor{},
		},
	}

	assert.Equal(t, expectedV1Prov.BuildDefinition.BuildType, v1Prov.BuildDefinition.BuildType, "BuildType mismatch")
	assert.Equal(t, expectedV1Prov.RunDetails.Builder.ID, v1Prov.RunDetails.Builder.ID, "Builder ID mismatch")
	assert.Equal(t, expectedV1Prov.RunDetails.BuildMetadata.InvocationID, v1Prov.RunDetails.BuildMetadata.InvocationID, "BuildMetadata InvocationID mismatch")
	assert.Equal(t, expectedV1Prov.RunDetails.BuildMetadata.StartedOn, v1Prov.RunDetails.BuildMetadata.StartedOn, "BuildMetadata StartedOn mismatch")
	assert.Equal(t, expectedV1Prov.RunDetails.BuildMetadata.FinishedOn, v1Prov.RunDetails.BuildMetadata.FinishedOn, "BuildMetadata FinishedOn mismatch")
	assert.Equal(t, expectedV1Prov.BuildDefinition.ResolvedDependencies, v1Prov.BuildDefinition.ResolvedDependencies, "BuildDefinition ResolvedDependencies mismatch")
}
