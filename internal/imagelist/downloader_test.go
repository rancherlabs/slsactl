package imagelist

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeImageName_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple image with tag",
			input:    "registry.example.com/myimage:v1.0.0",
			expected: "myimage_v1.0.0",
		},
		{
			name:     "image with nested path",
			input:    "registry.example.com/org/repo/image:latest",
			expected: "org_repo_image_latest",
		},
		{
			name:     "image with digest",
			input:    "registry.example.com/image@sha256:abc123def456789012345678901234567890123456789012345678901234",
			expected: "image_sha256_abc123def456789012345678901234567890123456789012345678901234",
		},
		{
			name:     "image with digest including registry",
			input:    "docker.io/library/nginx@sha256:abc123def456789012345678901234567890123456789012345678901234",
			expected: "library_nginx_sha256_abc123def456789012345678901234567890123456789012345678901234",
		},
		{
			name:     "rancher image",
			input:    "docker.io/rancher/rancher:v2.8.0",
			expected: "rancher_rancher_v2.8.0",
		},
		{
			name:     "library image",
			input:    "nginx:latest",
			expected: "library_nginx_latest",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := sanitizeImageName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSanitizeImageName_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "path traversal with double dots",
			input: "../../../etc/passwd:v1",
		},
		{
			name:  "path traversal in tag",
			input: "image:../../../etc/passwd",
		},
		{
			name:  "hidden path traversal",
			input: "registry.example.com/safe/../../../etc/passwd:v1",
		},
		{
			name:  "double dots without slashes",
			input: "registry.example.com/..image:v1",
		},
		{
			name:  "double dots in middle of name",
			input: "registry.example.com/my..image:v1",
		},
		{
			name:  "windows-style path traversal",
			input: `registry.example.com\..\..\etc\passwd:v1`,
		},
		{
			name:  "absolute path attempt",
			input: "/etc/passwd:v1",
		},
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "dot slash",
			input: "../",
		},
		{
			name:  "only special characters",
			input: "/../..:/../..",
		},
		{
			name:  "very long name",
			input: "registry.example.com/" + strings.Repeat("a", 1000) + ":v1",
		},
		{
			name:  "spaces in name",
			input: "registry.example.com/my image:v 1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := sanitizeImageName(tc.input)

			assert.NotContains(t, result, "..", "sanitized name contains path traversal sequence")
			assert.NotContains(t, result, "/", "sanitized name contains forward slash")
			assert.NotContains(t, result, "\\", "sanitized name contains backslash")
			assert.False(t, strings.HasPrefix(result, "."), "sanitized name starts with dot")

			baseDir := "/safe/output/dir"
			fullPath := filepath.Join(baseDir, result+"_sbom.json")
			cleanPath := filepath.Clean(fullPath)

			assert.True(t, strings.HasPrefix(cleanPath, baseDir+"/"),
				"path %q escapes base directory %q", cleanPath, baseDir)
		})
	}
}
