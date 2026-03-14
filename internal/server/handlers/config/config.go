// Package config provides configuration types for the HTTP server.
package config

// ServerConfig holds the HTTP server address and TLS certificate paths.
type ServerConfig struct {
	ServerAddr  string `json:"server_addr,omitempty"`
	TLSCertPath string `json:"tls_cert_path,omitempty"`
	TLSKeyPath  string `json:"tls_key_path,omitempty"`
}
