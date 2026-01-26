package landlock

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/landlock-lsm/go-landlock/landlock"
	"github.com/landlock-lsm/go-landlock/landlock/syscall"
)

const dirMode = 0o700

// EnforceOrDie checks whether or not to enforce the landlock policy, and if so,
// apply it. Any error will result in os.Exit.
func EnforceOrDie() {
	val, _ := os.LookupEnv("LANDLOCK_MODE")
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

	rules := []landlock.Rule{
		landlock.ROFiles(
			"/etc/resolv.conf", // DNS resolution.
		).IgnoreIfMissing(),
		landlock.RWDirs(rwDirs...),
		landlock.RODirs(
			"/proc/self",
			"/etc/ssl",                     // Root CA bundles to establish TLS.
			"/var/lib/ca-certificates",     // Root CA bundles to establish TLS.
			filepath.Join(home, ".docker"), // Docker config to access OCI/registries.
		).IgnoreIfMissing(),
	}

	if helper, ok := credentialHelper(home); ok {
		if val, ok := os.LookupEnv("LANDLOCK_CREDENTIAL_HELPER"); ok && strings.EqualFold(val, "true") {
			rules = append(rules, helper...)
		} else {
			fmt.Println("ERR: cannot use landlock with docker-credential helpers: disable landlock with LANDLOCK_MODE=off")
			os.Exit(1)
		}
	}

	err = cfg.RestrictPaths(rules...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to enforce landlock policies (requires Linux 5.13+): %v\n", err)
		if val == "on" {
			os.Exit(2)
		}
	}
}

func credentialHelper(home string) ([]landlock.Rule, bool) {
	path := filepath.Join(home, ".docker/config.json")
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}

	defer f.Close()

	var cfg dockerConfig
	d := json.NewDecoder(f)
	err = d.Decode(&cfg)
	if err != nil {
		return nil, false
	}

	if cfg.CredsStore == "" {
		return nil, false
	}

	fullPath, err := exec.LookPath("docker-credential-" + cfg.CredsStore)
	if err == nil {
		rules := []landlock.Rule{
			execFile(fullPath, "/bin/dbus-launch"),
			landlock.RODirs("/lib", "/lib64", "/usr/lib", "/usr/lib64", "/proc/self"),
			landlock.ROFiles("/etc/ld.so.cache", "/var/lib/dbus/machine-id", "/etc/machine-id").IgnoreIfMissing(),
			landlock.ROFiles("/dev/null", "/dev/zero"),
		}

		return rules, true
	}

	return nil, false
}

func execFile(path ...string) landlock.FSRule {
	return landlock.PathAccess(syscall.AccessFSExecute|syscall.AccessFSReadFile, path...)
}

type dockerConfig struct {
	CredsStore string `json:"credsStore"`
}

func ensureDirs(dirs []string) {
	for _, dir := range dirs {
		err := os.MkdirAll(dir, dirMode)
		if err != nil {
			slog.Error("failed to ensure dir", "path", dir, "error", err)
		}
	}
}
