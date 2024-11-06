package verify

import (
	"context"
	"crypto"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/verify"
)

const (
	timeout = 45 * time.Second
)

func Verify(imageName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	certIdentity, err := certIdentity(imageName)
	if err != nil {
		return err
	}

	fmt.Println("identity:", certIdentity)

	v := &verify.VerifyCommand{
		CertVerifyOptions: options.CertVerifyOptions{
			CertIdentity:   certIdentity,
			CertOidcIssuer: "https://token.actions.githubusercontent.com",
		},
		CheckClaims:   true,
		HashAlgorithm: crypto.SHA256,
		MaxWorkers:    5,
	}

	if strings.EqualFold(os.Getenv("DEBUG"), "true") {
		logs.Debug.SetOutput(os.Stderr)
	}

	return v.Exec(ctx, []string{imageName})
}

func certIdentity(imageName string) (string, error) {
	if len(imageName) < 5 {
		return "", fmt.Errorf("invalid image name: %q", imageName)
	}

	if strings.Contains(imageName, "@") {
		fmt.Println("warn: image name with digest is not supported, use tags only.")
		imageName = strings.Split(imageName, "@")[0]
	}

	d := strings.Split(imageName, ":")
	if len(d) < 2 || len(d[1]) == 0 {
		return "", fmt.Errorf("missing image tag: %q", imageName)
	}

	names := strings.Split(d[0], "/")
	if len(names) < 2 {
		return "", fmt.Errorf("unsupported image name: %q", imageName)
	}

	repo := strings.Join(names[len(names)-2:], "/")
	ref := strings.Replace(d[1], "-", "&#43;", 1)
	repo = overrideRepo(repo)

	indentity := fmt.Sprintf(
		"https://github.com/%s/.github/workflows/release.yml@refs/tags/%s", repo, ref)

	return indentity, nil
}

func overrideRepo(repo string) string {
	if strings.HasPrefix(repo, "rancher/rke2-") {
		return "rancher/rke2"
	}
	return repo
}
