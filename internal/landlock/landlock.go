package landlock

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/landlock-lsm/go-landlock/landlock"
)

// EnforceOrDie checks whether or not to enforce the landlock policy, and if so,
// apply it. Any error will result in os.Exit.
func EnforceOrDie() {
	val, ok := os.LookupEnv("LANDLOCK_MODE")
	if !ok {
		return
	}

	cfg := landlock.V5

	switch {
	case strings.EqualFold(val, "on"):
		slog.Debug("landlock enabled") //nolint: noctx
	case strings.EqualFold(val, "besteffort"):
		cfg = cfg.BestEffort()
		slog.Debug("landlock set to best effort") //nolint: noctx
	default:
		slog.Debug("landlock disabled") //nolint: noctx
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("failed to get user home dir: %v", err)
		os.Exit(1)
	}

	// We can't really restrict the network access at present
	err = cfg.RestrictPaths(
		landlock.RWDirs(filepath.Join(home, ".sigstore")), // Sigstore TUF DB.
		landlock.RODirs(
			"/proc/self",
		),
		landlock.ROFiles(
			"/etc/resolv.conf",                         // DNS resolution.
			"/etc/ssl/ca-bundle.pem",                   // Root CA bundles to establish TLS.
			filepath.Join(home, ".docker/config.json"), // Docker config to access OCI/registries.
		),
	)
	if err != nil {
		fmt.Printf("failed to enforce landlock policies (requires Linux 5.13+): %v", err)
		os.Exit(2)
	}
}
