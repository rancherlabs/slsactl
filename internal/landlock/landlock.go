package landlock

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/landlock-lsm/go-landlock/landlock"
)

const dirMode = 0o700

// EnforceOrDie checks whether or not to enforce the landlock policy, and if so,
// apply it. Any error will result in os.Exit.
func EnforceOrDie() {
	val, ok := os.LookupEnv("LANDLOCK_MODE")
	if !ok || val == "" {
		val = "besteffort"
	}

	cfg := landlock.V5

	switch {
	case strings.EqualFold(val, "on"):
		slog.Debug("landlock enabled")
	case strings.EqualFold(val, "besteffort"):
		cfg = cfg.BestEffort()
		slog.Debug("landlock set to best effort")
	default:
		slog.Debug("landlock disabled")
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("failed to get user home dir: %v", err)
		os.Exit(1)
	}

	rwDirs := []string{
		filepath.Join(home, ".sigstore"),         // Sigstore TUF DB.
		filepath.Join(home, ".docker", "buildx"), // Image artefacts handling.
	}
	ensureDirs(rwDirs)

	cwd, err := os.Getwd()
	if err == nil {
		rwDirs = append(rwDirs, cwd) // Needs write access to CWD for "product verify" subcommand
	}

	// We can't really restrict the network access at present
	err = cfg.RestrictPaths(
		landlock.RWDirs(rwDirs...),
		landlock.RODirs(
			"/proc/self",
			"/etc/ssl",                     // Root CA bundles to establish TLS.
			filepath.Join(home, ".docker"), // Docker config to access OCI/registries.
		),
		landlock.ROFiles(
			"/etc/resolv.conf", // DNS resolution.
		),
	)
	if err != nil {
		fmt.Printf("failed to enforce landlock policies (requires Linux 5.13+): %v\n", err)
		if val == "on" {
			os.Exit(2)
		}
	}
}

func ensureDirs(dirs []string) {
	for _, dir := range dirs {
		err := os.MkdirAll(dir, dirMode)
		if err != nil {
			slog.Error("failed to ensure dir", "path", dir, "error", err)
		}
	}
}
