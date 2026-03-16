package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	logconf "github.com/ar4ie13/gophkeeper/internal/logger/config"
	authconf "github.com/ar4ie13/gophkeeper/internal/server/auth/config"
	serverconf "github.com/ar4ie13/gophkeeper/internal/server/handlers/config"
	pgconf "github.com/ar4ie13/gophkeeper/internal/server/repository/db/postgresql/config"
	serviceconf "github.com/ar4ie13/gophkeeper/internal/server/service/config"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Config aggregates all server configuration settings from various subsystems.
type Config struct {
	ServerConf  serverconf.ServerConfig `json:"server_conf,omitempty"`
	LogConf     logconf.LogLevel        `json:"log_conf,omitempty"`
	AuthConf    authconf.Config         `json:"auth_conf,omitempty"`
	ServiceConf serviceconf.ServiceConf `json:"service_conf,omitempty"`
	PGConf      pgconf.PGConf           `json:"pg_conf,omitempty"`
	ConfigFile  string                  `json:"config_file,omitempty"`
}

// NewConfig creates a new Config instance with defaults, loads configuration from file and environment variables.
func NewConfig() *Config {
	c := &Config{
		ServerConf: serverconf.ServerConfig{
			ServerAddr:  "localhost:8080",
			TLSCertPath: "cert.pem",
			TLSKeyPath:  "private.pem",
		},

		LogConf: logconf.LogLevel{
			Level: zerolog.DebugLevel,
		},
		AuthConf: authconf.Config{
			TokenExpiration: time.Hour * 24,
			PasswordLen:     8,
		},
		ConfigFile: "config.json",
	}

	if err := c.loadConfigFile(); err != nil {
		log.Fatal().Err(err).Msg("Failed to load config file")
	}

	_ = godotenv.Load()

	c.loadEnv()

	return c
}

// loadEnv loads configuration values from environment variables, overriding file-based settings.
func (c *Config) loadEnv() {
	// PG DSN
	if databaseDSN := os.Getenv("DATABASE_DSN"); databaseDSN != "" {
		c.PGConf.DatabaseDSN = databaseDSN

	}

	// Master key for encryption
	if masterKey := os.Getenv("MASTER_KEY"); masterKey != "" {
		c.ServiceConf.MasterKey = masterKey
	}

	// Secret key for authentication and authorization
	if secretKey := os.Getenv("SECRET_KEY"); secretKey != "" {
		c.AuthConf.SecretKey = secretKey
	}
}

// loadConfigFile loads and merges configuration from a JSON file if it exists.
func (c *Config) loadConfigFile() error {
	file, err := os.ReadFile(c.ConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var fc Config
	if err = json.Unmarshal(file, &fc); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	mergeStr(&c.ServerConf.ServerAddr, fc.ServerConf.ServerAddr)
	mergeStr(&c.ServerConf.TLSKeyPath, fc.ServerConf.TLSKeyPath)
	mergeStr(&c.ServerConf.TLSCertPath, fc.ServerConf.TLSCertPath)
	mergeInt(&c.AuthConf.PasswordLen, fc.AuthConf.PasswordLen)
	mergeDuration(&c.AuthConf.TokenExpiration, fc.AuthConf.TokenExpiration)

	if fc.LogConf.Level != zerolog.NoLevel {
		c.LogConf.Level = fc.LogConf.Level
	}

	return nil
}

// mergeStr updates dst with src if src is not empty.
func mergeStr(dst *string, src string) {
	if src != "" {
		*dst = src
	}
}

// mergeBool updates dst with src if src is true.
func mergeBool(dst *bool, src bool) {
	if src {
		*dst = src
	}
}

// mergeDuration updates dst with src (converted to hours) if src is not zero.
func mergeDuration(dst *time.Duration, src time.Duration) {
	if src != 0 {
		*dst = src * time.Hour
	}
}

// mergeInt updates dst with src if src is not zero.
func mergeInt(dst *int, src int) {
	if src != 0 {
		*dst = src
	}
}
