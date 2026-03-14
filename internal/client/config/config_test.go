package config

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetFlags resets the flag.CommandLine before each test so flags can be
// re-registered without "flag redefined" panics.
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

func TestNewConfig_Defaults(t *testing.T) {
	resetFlags()
	os.Args = []string{"cmd"}

	c := NewConfig()

	assert.Equal(t, "http://localhost:8080", c.ServerURL)
	assert.Nil(t, c.CaCert)
}

func TestNewConfig_ServerURLFlag(t *testing.T) {
	resetFlags()
	os.Args = []string{"cmd", "-server", "https://example.com:9443"}

	c := NewConfig()

	assert.Equal(t, "https://example.com:9443", c.ServerURL)
}

func TestNewConfig_CaCertFlag(t *testing.T) {
	certContent := []byte("-----BEGIN CERTIFICATE-----\nfake cert\n-----END CERTIFICATE-----\n")
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "ca.pem")
	require.NoError(t, os.WriteFile(certPath, certContent, 0644))

	resetFlags()
	os.Args = []string{"cmd", "-ca-cert", certPath}

	c := NewConfig()

	assert.Equal(t, certContent, c.CaCert)
}

func TestNewConfig_NoCaCert(t *testing.T) {
	resetFlags()
	os.Args = []string{"cmd"}

	c := NewConfig()

	assert.Empty(t, c.CaCert)
}

func TestNewConfig_BuildInfo(t *testing.T) {
	// Save originals and restore after test.
	origVersion := buildVersion
	origDate := buildDate
	origCommit := buildCommit
	defer func() {
		buildVersion = origVersion
		buildDate = origDate
		buildCommit = origCommit
	}()

	buildVersion = "1.2.3"
	buildDate = "2026-03-14"
	buildCommit = "abc123"

	resetFlags()
	os.Args = []string{"cmd"}

	c := NewConfig()

	assert.Equal(t, "1.2.3", c.Version)
	assert.Equal(t, "2026-03-14", c.BuildDate)
	assert.Equal(t, "abc123", c.BuildCommit)
}

func TestNewConfig_DefaultBuildInfo(t *testing.T) {
	// Ensure package-level defaults ("N/A") are reflected in the struct.
	origVersion := buildVersion
	origDate := buildDate
	origCommit := buildCommit
	defer func() {
		buildVersion = origVersion
		buildDate = origDate
		buildCommit = origCommit
	}()

	buildVersion = "N/A"
	buildDate = "N/A"
	buildCommit = "N/A"

	resetFlags()
	os.Args = []string{"cmd"}

	c := NewConfig()

	assert.Equal(t, "N/A", c.Version)
	assert.Equal(t, "N/A", c.BuildDate)
	assert.Equal(t, "N/A", c.BuildCommit)
}

func TestNewConfig_ServerURLAndCaCert(t *testing.T) {
	certContent := []byte("cert data")
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "ca.pem")
	require.NoError(t, os.WriteFile(certPath, certContent, 0644))

	resetFlags()
	os.Args = []string{"cmd", "-server", "https://prod.example.com:443", "-ca-cert", certPath}

	c := NewConfig()

	assert.Equal(t, "https://prod.example.com:443", c.ServerURL)
	assert.Equal(t, certContent, c.CaCert)
}
