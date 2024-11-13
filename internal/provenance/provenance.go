package provenance

import (
	v02 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	v1 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
)

type InternalParameters struct {
	Platform      string `json:"platform,omitempty"`
	Trigger       string `json:"trigger,omitempty"`
	InvocationUri string `json:"invocationUri,omitempty"`
}

type BuildKitProvenance02 struct {
	LinuxAmd64 *ArchProvenance          `json:"linux/amd64,omitempty"`
	LinuxArm64 *ArchProvenance          `json:"linux/arm64,omitempty"`
	SLSA       *v02.ProvenancePredicate `json:"SLSA,omitempty"`
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

func ConvertV02ToV1(v02Prov v02.ProvenancePredicate, override *v1.ProvenancePredicate) v1.ProvenancePredicate {
	prov := v1.ProvenancePredicate{
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
				StartedOn:    v02Prov.Metadata.BuildStartedOn,
				FinishedOn:   v02Prov.Metadata.BuildFinishedOn,
				InvocationID: v02Prov.Metadata.BuildInvocationID,
			},
			Byproducts: []v1.ResourceDescriptor{},
		},
	}

	deps := make([]v1.ResourceDescriptor, 0, len(v02Prov.Materials))
	for _, m := range v02Prov.Materials {
		deps = append(deps, v1.ResourceDescriptor{
			URI:    m.URI,
			Digest: m.Digest,
		})
	}

	prov.BuildDefinition.ResolvedDependencies = deps

	if override != nil {
		if override.RunDetails.Builder.ID != "" {
			prov.RunDetails.Builder.ID = override.RunDetails.Builder.ID
		}
		if len(override.BuildDefinition.ResolvedDependencies) > 0 {
			prov.BuildDefinition.ResolvedDependencies =
				append(prov.BuildDefinition.ResolvedDependencies,
					override.BuildDefinition.ResolvedDependencies...)
		}
		if override.BuildDefinition.InternalParameters != nil {
			prov.BuildDefinition.InternalParameters = override.BuildDefinition.InternalParameters
		}
	}

	return prov
}
