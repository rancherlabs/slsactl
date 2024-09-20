package provenance

import (
	v02 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	v1 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
)

type BuildKitProvenance02 struct {
	LinuxAmd64 *ArchProvenance `json:"linux/amd64,omitempty"`
	LinuxArm64 *ArchProvenance `json:"linux/arm64,omitempty"`
}

type ArchProvenance struct {
	SLSA v02.ProvenancePredicate `json:"SLSA,omitempty"`
}

type SLSAV1Provenance struct {
	LinuxAmd64 *ArchProvenanceV1 `json:"linux/amd64,omitempty"`
	LinuxArm64 *ArchProvenanceV1 `json:"linux/arm64,omitempty"`
}
type ArchProvenanceV1 struct {
	SLSA v1.ProvenancePredicate `json:"SLSA,omitempty"`
}

func ConvertV02ToV1(v02Prov v02.ProvenancePredicate) v1.ProvenancePredicate {
	return v1.ProvenancePredicate{
		BuildDefinition: v1.ProvenanceBuildDefinition{
			BuildType:          v02Prov.BuildType,
			ExternalParameters: v02Prov.Invocation.Parameters,
			InternalParameters: v02Prov.Invocation.Environment,
		},
		RunDetails: v1.ProvenanceRunDetails{
			Builder: v1.Builder{
				ID: v02Prov.Invocation.ConfigSource.URI,
			},
			BuildMetadata: v1.BuildMetadata{
				StartedOn:  v02Prov.Metadata.BuildStartedOn,
				FinishedOn: v02Prov.Metadata.BuildFinishedOn,
			},
			Byproducts: []v1.ResourceDescriptor{},
		},
	}
}
