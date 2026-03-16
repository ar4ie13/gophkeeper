package config

import (
	"flag"
	"fmt"
	"os"
)

// Build information variables set during compilation via ldflags.
var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

// Config holds client configuration including server URL and TLS certificate.
type Config struct {
	ServerURL   string
	CaCert      []byte
	Version     string
	BuildDate   string
	BuildCommit string
}

// NewConfig creates a new Config instance from command-line flags.
func NewConfig() *Config {
	fmt.Println("Build version: " + buildVersion)
	fmt.Println("Build date: " + buildDate)
	fmt.Println("Build commit: " + buildCommit)
	var config Config
	config.Version = buildVersion
	config.BuildDate = buildDate
	config.BuildCommit = buildCommit
	serverURL := flag.String("server", "http://localhost:8080", "Server base URL")
	caCertPath := flag.String("ca-cert", "", "Path to a PEM-encoded CA certificate for TLS verification")
	flag.Parse()
	config.ServerURL = *serverURL

	// Read custom CA certificate if provided.
	var caCert []byte
	if *caCertPath != "" {
		var err error
		caCert, err = os.ReadFile(*caCertPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot read CA certificate %q: %v\n", *caCertPath, err)
			os.Exit(1)
		}
	}
	config.CaCert = caCert

	return &config
}
