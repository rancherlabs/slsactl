package imagelist

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRegistry(t *testing.T, referrers bool) string {
	t.Helper()
	s := httptest.NewServer(registry.New(registry.WithReferrersSupport(referrers)))
	t.Cleanup(s.Close)
	return strings.TrimPrefix(s.URL, "http://")
}

// pushSignedImage pushes an image tagged v1 and a cosign v3 style referrer
// (OCI manifest with subject) pointing at its digest.
func pushSignedImage(t *testing.T, repo string) (string, v1.Hash) {
	t.Helper()

	return pushReferrerImage(t, repo, "application/vnd.dev.sigstore.bundle.v0.3+json")
}

// pushReferrerImage pushes an image tagged v1 and a referrer of the given artifact type.
func pushReferrerImage(t *testing.T, repo, artifactType string) (string, v1.Hash) {
	t.Helper()

	img, err := random.Image(64, 1)
	require.NoError(t, err)
	imgRef := repo + ":v1"
	require.NoError(t, crane.Push(img, imgRef))

	return imgRef, pushArtifactReferrer(t, repo, img, artifactType)
}

func pushArtifactReferrer(t *testing.T, repo string, img v1.Image, artifactType string) v1.Hash {
	t.Helper()

	if strings.HasPrefix(artifactType, sigstoreBundleTypePrefix) {
		return pushBundleReferrer(t, repo, img, artifactType,
			[]byte(`{"messageSignature":{}}`),
			map[string]string{sigstoreBundleContentAnnotation: "message-signature"})
	}

	d, err := img.Digest()
	require.NoError(t, err)
	mt, err := img.MediaType()
	require.NoError(t, err)
	sz, err := img.Size()
	require.NoError(t, err)
	sig := mutate.MediaType(empty.Image, types.OCIManifestSchema1)
	sig = mutate.ConfigMediaType(sig, types.MediaType(artifactType))
	sig = mutate.Subject(sig, v1.Descriptor{MediaType: mt, Digest: d, Size: sz}).(v1.Image)
	sigDigest, err := sig.Digest()
	require.NoError(t, err)

	r, err := name.ParseReference(fmt.Sprintf("%s@%s", repo, sigDigest))
	require.NoError(t, err)
	require.NoError(t, remote.Write(r, sig))

	return sigDigest
}

func pushBundleReferrer(t *testing.T, repo string, img v1.Image, artifactType string, bundle []byte, annotations map[string]string) v1.Hash {
	t.Helper()

	d, err := img.Digest()
	require.NoError(t, err)
	mt, err := img.MediaType()
	require.NoError(t, err)
	sz, err := img.Size()
	require.NoError(t, err)

	sig := mutate.MediaType(empty.Image, types.OCIManifestSchema1)
	sig = mutate.ConfigMediaType(sig, types.MediaType(artifactType))
	sig, err = mutate.Append(sig, mutate.Addendum{
		Layer: static.NewLayer(bundle, types.MediaType(artifactType)),
	})
	require.NoError(t, err)
	sig = mutate.Annotations(sig, annotations).(v1.Image)
	sig = mutate.Subject(sig, v1.Descriptor{MediaType: mt, Digest: d, Size: sz}).(v1.Image)

	sigDigest, err := sig.Digest()
	require.NoError(t, err)
	r, err := name.ParseReference(fmt.Sprintf("%s@%s", repo, sigDigest))
	require.NoError(t, err)
	require.NoError(t, remote.Write(r, sig))

	return sigDigest
}

func TestCopySignatureReferrers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		srcReferrers bool
		dstReferrers bool
	}{
		{name: "src and dst with Referrers API", srcReferrers: true, dstReferrers: true},
		{name: "src with Referrers API, dst with tag fallback", srcReferrers: true, dstReferrers: false},
		{name: "src with tag fallback, dst with Referrers API", srcReferrers: false, dstReferrers: true},
		{name: "src and dst with tag fallback", srcReferrers: false, dstReferrers: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			src := newRegistry(t, tc.srcReferrers) + "/rancher/app"
			dst := newRegistry(t, tc.dstReferrers) + "/rancher/app"

			srcImg, sigDigest := pushSignedImage(t, src)
			require.NoError(t, CopySignature(context.Background(), srcImg, dst+":v1", true))

			digest, err := crane.Digest(dst + ":v1")
			require.NoError(t, err)
			subject, err := name.NewDigest(dst + "@" + digest)
			require.NoError(t, err)

			idx, err := remote.Referrers(subject)
			require.NoError(t, err)
			m, err := idx.IndexManifest()
			require.NoError(t, err)
			require.Len(t, m.Manifests, 1)
			assert.Equal(t, sigDigest, m.Manifests[0].Digest)
		})
	}
}

func TestCopySignatureNoSignatures(t *testing.T) {
	t.Parallel()

	src := newRegistry(t, true) + "/rancher/app"
	dst := newRegistry(t, true) + "/rancher/app"

	img, err := random.Image(64, 1)
	require.NoError(t, err)
	require.NoError(t, crane.Push(img, src+":v1"))

	err = CopySignature(context.Background(), src+":v1", dst+":v1", false)
	require.ErrorIs(t, err, ErrNoSignaturesFound)
}

func TestCopySignatureIgnoresNonSignatureReferrers(t *testing.T) {
	t.Parallel()

	src := newRegistry(t, true) + "/rancher/app"
	dst := newRegistry(t, true) + "/rancher/app"

	srcImg, _ := pushReferrerImage(t, src, "application/spdx+json")

	err := CopySignature(context.Background(), srcImg, dst+":v1", true)
	require.ErrorIs(t, err, ErrNoSignaturesFound)
	assert.Equal(t, 0, referrerCount(t, dst, "v1"), "non-signature referrers must not be copied")
}

func TestCopySignatureIgnoresAttestationBundle(t *testing.T) {
	t.Parallel()

	src := newRegistry(t, true) + "/rancher/app"
	dst := newRegistry(t, true) + "/rancher/app"

	srcImg := pushPlainImage(t, src)
	image, err := crane.Pull(srcImg)
	require.NoError(t, err)
	pushBundleReferrer(t, src, image, sigstoreBundleTypePrefix+".v0.3+json",
		[]byte(`{"dsseEnvelope":{"payload":"eyJwcmVkaWNhdGVUeXBlIjoiaHR0cHM6Ly9zbHNhLmRldi9wcm92ZW5hbmNlL3YxIn0="}}`),
		map[string]string{
			sigstoreBundleContentAnnotation:   "dsse-envelope",
			sigstoreBundlePredicateAnnotation: "https://slsa.dev/provenance/v1",
		})

	err = CopySignature(context.Background(), srcImg, dst+":v1", true)
	require.ErrorIs(t, err, ErrNoSignaturesFound)
	assert.Equal(t, 0, referrerCount(t, dst, "v1"), "attestation bundles must not be copied as signatures")
}

func TestCopySignatureFallbackTagFiltersNonSignatures(t *testing.T) {
	t.Parallel()

	src := newRegistry(t, false) + "/rancher/app"
	dst := newRegistry(t, true) + "/rancher/app"

	srcImg, _ := pushReferrerImage(t, src, "application/spdx+json")

	err := CopySignature(context.Background(), srcImg, dst+":v1", true)
	require.ErrorIs(t, err, ErrNoSignaturesFound)
	assert.Equal(t, 0, referrerCount(t, dst, "v1"), "non-signature referrers must not be copied")
}

func TestCopySignatureFallbackTagFiltersMixedReferrers(t *testing.T) {
	t.Parallel()

	src := newRegistry(t, false) + "/rancher/app"
	dst := newRegistry(t, true) + "/rancher/app"

	srcImg := pushPlainImage(t, src)
	image, err := crane.Pull(srcImg)
	require.NoError(t, err)
	pushArtifactReferrer(t, src, image, "application/spdx+json")
	sigDigest := pushBundleReferrer(t, src, image, sigstoreBundleTypePrefix+".v0.3+json",
		[]byte(`{"messageSignature":{}}`), nil)

	require.NoError(t, CopySignature(context.Background(), srcImg, dst+":v1", true))
	assert.Equal(t, 1, referrerCount(t, dst, "v1"), "only signature referrers must be copied")

	digest, err := crane.Digest(dst + ":v1")
	require.NoError(t, err)
	subject, err := name.NewDigest(dst + "@" + digest)
	require.NoError(t, err)
	idx, err := remote.Referrers(subject)
	require.NoError(t, err)
	manifest, err := idx.IndexManifest()
	require.NoError(t, err)
	require.Len(t, manifest.Manifests, 1)
	assert.Equal(t, sigDigest, manifest.Manifests[0].Digest)
}

func TestCopySignatureReferrerTransferFailure(t *testing.T) {
	t.Parallel()

	src := newRegistry(t, true) + "/rancher/app"
	srcImg, _ := pushSignedImage(t, src)

	err := CopySignature(context.Background(), srcImg, "127.0.0.1:1/rancher/app:v1", false)
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNoSignaturesFound)
}

// pushTaggedSignature pushes a random image tagged sha256-<hex of imgRef digest><suffix>
// to repo, the way cosign publishes legacy signatures (suffix ".sig") or the
// tag used by the no-suffix format (suffix ""). It returns the tag and its digest.
func pushTaggedSignature(t *testing.T, repo, imgRef, suffix string) (string, string) {
	t.Helper()

	d, err := crane.Digest(imgRef)
	require.NoError(t, err)
	tag := "sha256-" + strings.TrimPrefix(d, "sha256:") + suffix

	sig, err := random.Image(32, 1)
	require.NoError(t, err)
	require.NoError(t, crane.Push(sig, repo+":"+tag))

	sigDigest, err := crane.Digest(repo + ":" + tag)
	require.NoError(t, err)
	return tag, sigDigest
}

func pushPlainImage(t *testing.T, repo string) string {
	t.Helper()

	img, err := random.Image(64, 1)
	require.NoError(t, err)
	require.NoError(t, crane.Push(img, repo+":v1"))
	return repo + ":v1"
}

func referrerCount(t *testing.T, repo, tag string) int {
	t.Helper()

	digest, err := crane.Digest(repo + ":" + tag)
	require.NoError(t, err)
	subject, err := name.NewDigest(repo + "@" + digest)
	require.NoError(t, err)
	idx, err := remote.Referrers(subject)
	require.NoError(t, err)
	m, err := idx.IndexManifest()
	require.NoError(t, err)
	return len(m.Manifests)
}

func TestCopySignatureTagFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		suffix string
	}{
		{name: "legacy .sig tag", suffix: ".sig"},
		{name: "tag without .sig suffix", suffix: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			src := newRegistry(t, true) + "/rancher/app"
			dst := newRegistry(t, true) + "/rancher/app"

			srcImg := pushPlainImage(t, src)
			tag, sigDigest := pushTaggedSignature(t, src, srcImg, tc.suffix)

			require.NoError(t, CopySignature(context.Background(), srcImg, dst+":v1", true))

			got, err := crane.Digest(dst + ":" + tag)
			require.NoError(t, err)
			assert.Equal(t, sigDigest, got)
		})
	}
}

// TestCopySignatureFormatPrecedence documents the current behavior: the first
// format found wins and the remaining ones are not copied.
func TestCopySignatureFormatPrecedence(t *testing.T) {
	t.Parallel()

	t.Run("legacy .sig wins over plain tag and referrers", func(t *testing.T) {
		t.Parallel()

		src := newRegistry(t, true) + "/rancher/app"
		dst := newRegistry(t, true) + "/rancher/app"

		srcImg, _ := pushSignedImage(t, src)
		legacyTag, legacyDigest := pushTaggedSignature(t, src, srcImg, ".sig")
		plainTag, _ := pushTaggedSignature(t, src, srcImg, "")

		require.NoError(t, CopySignature(context.Background(), srcImg, dst+":v1", true))

		got, err := crane.Digest(dst + ":" + legacyTag)
		require.NoError(t, err)
		assert.Equal(t, legacyDigest, got)

		_, err = crane.Digest(dst + ":" + plainTag)
		assert.Error(t, err, "plain tag must not be copied when .sig exists")
		assert.Equal(t, 0, referrerCount(t, dst, "v1"), "referrers must not be copied when .sig exists")
	})

	t.Run("plain tag wins over referrers", func(t *testing.T) {
		t.Parallel()

		src := newRegistry(t, true) + "/rancher/app"
		dst := newRegistry(t, true) + "/rancher/app"

		srcImg, _ := pushSignedImage(t, src)
		plainTag, plainDigest := pushTaggedSignature(t, src, srcImg, "")

		require.NoError(t, CopySignature(context.Background(), srcImg, dst+":v1", true))

		got, err := crane.Digest(dst + ":" + plainTag)
		require.NoError(t, err)
		assert.Equal(t, plainDigest, got)
		assert.Equal(t, 0, referrerCount(t, dst, "v1"), "referrers must not be copied when the plain tag exists")
	})
}

// leafErrors flattens wrapped and joined errors into their leaves.
func leafErrors(err error) []error {
	var multi interface{ Unwrap() []error }
	if errors.As(err, &multi) {
		out := make([]error, 0, 4)
		for _, c := range multi.Unwrap() {
			out = append(out, leafErrors(c)...)
		}

		return out
	}

	var single interface{ Unwrap() error }
	if errors.As(err, &single) {
		return leafErrors(single.Unwrap())
	}

	return []error{err}
}

// TestCopySignatureNoSignaturesErrors checks that the aggregated error carries
// one distinct error per format tried (legacy tag, plain tag, referrers).
func TestCopySignatureNoSignaturesErrors(t *testing.T) {
	t.Parallel()

	src := newRegistry(t, true) + "/rancher/app"
	dst := newRegistry(t, true) + "/rancher/app"
	srcImg := pushPlainImage(t, src)

	err := CopySignature(context.Background(), srcImg, dst+":v1", false)
	require.ErrorIs(t, err, ErrNoSignaturesFound)

	var causes []error
	for _, e := range leafErrors(err) {
		if !errors.Is(e, ErrNoSignaturesFound) || e.Error() != ErrNoSignaturesFound.Error() {
			causes = append(causes, e)
		}
	}
	require.Len(t, causes, 3, "expected one error per signature format")

	seen := map[string]bool{}
	for _, c := range causes {
		assert.False(t, seen[c.Error()], "duplicated error: %v", c)
		seen[c.Error()] = true
	}

	assert.Contains(t, causes[0].Error(), ".sig")
	assert.NotContains(t, causes[1].Error(), ".sig")
	assert.Contains(t, causes[2].Error(), "no signature referrers found")
}
