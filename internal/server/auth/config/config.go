package config

import "time"

// Config object for authentication service
type Config struct {
	SecretKey       string        `json:"secret_key,omitempty"`
	TokenExpiration time.Duration `json:"token_expiration,omitempty"`
	PasswordLen     int           `json:"password_length,omitempty"`
}
