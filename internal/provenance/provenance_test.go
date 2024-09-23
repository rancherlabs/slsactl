package provenance

import (
	"encoding/json"
	"testing"
	"time"

	_ "embed"

	v02 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	v1 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed security-scan.v0.2
var v02Data []byte

func TestConvertV02ToV1(t *testing.T) {
	var v02Prov v02.ProvenancePredicate
	err := json.Unmarshal(v02Data, &v02Prov)
	require.NoError(t, err, "Failed to unmarshal v0.2 data")

	v1Prov := ConvertV02ToV1(v02Prov)

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
		},
		RunDetails: v1.ProvenanceRunDetails{
			Builder: v1.Builder{
				ID: "",
			},
			BuildMetadata: v1.BuildMetadata{
				StartedOn:  &started,
				FinishedOn: &finished,
			},
			Byproducts: []v1.ResourceDescriptor{},
		},
	}

	assert.Equal(t, expectedV1Prov.BuildDefinition.BuildType, v1Prov.BuildDefinition.BuildType, "BuildType mismatch")
	assert.Equal(t, expectedV1Prov.RunDetails.Builder.ID, v1Prov.RunDetails.Builder.ID, "Builder ID mismatch")
	assert.Equal(t, expectedV1Prov.RunDetails.BuildMetadata.StartedOn, v1Prov.RunDetails.BuildMetadata.StartedOn, "BuildMetadata StartedOn mismatch")
	assert.Equal(t, expectedV1Prov.RunDetails.BuildMetadata.FinishedOn, v1Prov.RunDetails.BuildMetadata.FinishedOn, "BuildMetadata FinishedOn mismatch")
}
