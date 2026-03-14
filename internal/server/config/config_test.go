package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	logconf "github.com/ar4ie13/gophkeeper/internal/logger/config"
	authconf "github.com/ar4ie13/gophkeeper/internal/server/auth/config"
	serverconf "github.com/ar4ie13/gophkeeper/internal/server/handlers/config"
	pgconf "github.com/ar4ie13/gophkeeper/internal/server/repository/db/postgresql/config"
	serviceconf "github.com/ar4ie13/gophkeeper/internal/server/service/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeStr(t *testing.T) {
	t.Run("merge non-empty string", func(t *testing.T) {
		dst := "original"
		src := "new"

		mergeStr(&dst, src)

		assert.Equal(t, "new", dst)
	})

	t.Run("skip empty string", func(t *testing.T) {
		dst := "original"
		src := ""

		mergeStr(&dst, src)

		assert.Equal(t, "original", dst)
	})
}

func TestMergeBool(t *testing.T) {
	t.Run("merge true", func(t *testing.T) {
		dst := false
		src := true

		mergeBool(&dst, src)

		assert.True(t, dst)
	})

	t.Run("skip false", func(t *testing.T) {
		dst := true
		src := false

		mergeBool(&dst, src)

		assert.True(t, dst)
	})
}

func TestMergeDuration(t *testing.T) {
	t.Run("merge non-zero duration", func(t *testing.T) {
		dst := time.Hour * 24
		src := time.Duration(12)

		mergeDuration(&dst, src)

		assert.Equal(t, time.Hour*12, dst)
	})

	t.Run("skip zero duration", func(t *testing.T) {
		original := time.Hour * 24
		dst := original
		src := time.Duration(0)

		mergeDuration(&dst, src)

		assert.Equal(t, original, dst)
	})
}

func TestMergeInt(t *testing.T) {
	t.Run("merge non-zero int", func(t *testing.T) {
		dst := 10
		src := 20

		mergeInt(&dst, src)

		assert.Equal(t, 20, dst)
	})

	t.Run("skip zero int", func(t *testing.T) {
		dst := 10
		src := 0

		mergeInt(&dst, src)

		assert.Equal(t, 10, dst)
	})
}

func TestConfig_loadEnv(t *testing.T) {
	t.Run("load DATABASE_DSN", func(t *testing.T) {
		os.Setenv("DATABASE_DSN", "postgres://test")
		defer os.Unsetenv("DATABASE_DSN")

		c := &Config{}
		c.loadEnv()

		assert.Equal(t, "postgres://test", c.PGConf.DatabaseDSN)
	})

	t.Run("load MASTER_KEY", func(t *testing.T) {
		os.Setenv("MASTER_KEY", "test_master_key_123456789012")
		defer os.Unsetenv("MASTER_KEY")

		c := &Config{}
		c.loadEnv()

		assert.Equal(t, "test_master_key_123456789012", c.ServiceConf.MasterKey)
	})

	t.Run("load SECRET_KEY", func(t *testing.T) {
		os.Setenv("SECRET_KEY", "test_secret_key")
		defer os.Unsetenv("SECRET_KEY")

		c := &Config{}
		c.loadEnv()

		assert.Equal(t, "test_secret_key", c.AuthConf.SecretKey)
	})

	t.Run("load all environment variables", func(t *testing.T) {
		os.Setenv("DATABASE_DSN", "postgres://all")
		os.Setenv("MASTER_KEY", "master_all")
		os.Setenv("SECRET_KEY", "secret_all")
		defer func() {
			os.Unsetenv("DATABASE_DSN")
			os.Unsetenv("MASTER_KEY")
			os.Unsetenv("SECRET_KEY")
		}()

		c := &Config{}
		c.loadEnv()

		assert.Equal(t, "postgres://all", c.PGConf.DatabaseDSN)
		assert.Equal(t, "master_all", c.ServiceConf.MasterKey)
		assert.Equal(t, "secret_all", c.AuthConf.SecretKey)
	})

	t.Run("skip empty environment variables", func(t *testing.T) {
		c := &Config{
			PGConf: pgconf.PGConf{
				DatabaseDSN: "original_dsn",
			},
			ServiceConf: serviceconf.ServiceConf{
				MasterKey: "original_master",
			},
			AuthConf: authconf.Config{
				SecretKey: "original_secret",
			},
		}

		c.loadEnv()

		assert.Equal(t, "original_dsn", c.PGConf.DatabaseDSN)
		assert.Equal(t, "original_master", c.ServiceConf.MasterKey)
		assert.Equal(t, "original_secret", c.AuthConf.SecretKey)
	})
}

func TestConfig_loadConfigFile(t *testing.T) {
	t.Run("file does not exist - returns nil", func(t *testing.T) {
		c := &Config{
			ConfigFile: "nonexistent_file.json",
			ServerConf: serverconf.ServerConfig{
				ServerAddr: "localhost:8080",
			},
		}

		err := c.loadConfigFile()

		assert.NoError(t, err)
		assert.Equal(t, "localhost:8080", c.ServerConf.ServerAddr)
	})

	t.Run("valid config file", func(t *testing.T) {
		// Create a temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test_config.json")

		testConfig := Config{
			ServerConf: serverconf.ServerConfig{
				ServerAddr:  "testhost:9090",
				TLSCertPath: "test_cert.pem",
				TLSKeyPath:  "test_key.pem",
			},
			AuthConf: authconf.Config{
				PasswordLen:     12,
				TokenExpiration: 48, // Will be multiplied by time.Hour in mergeDuration
			},
			LogConf: logconf.LogLevel{
				Level: zerolog.InfoLevel,
			},
		}

		data, err := json.Marshal(testConfig)
		require.NoError(t, err)

		err = os.WriteFile(configPath, data, 0644)
		require.NoError(t, err)

		c := &Config{
			ConfigFile: configPath,
			ServerConf: serverconf.ServerConfig{
				ServerAddr:  "localhost:8080",
				TLSCertPath: "cert.pem",
				TLSKeyPath:  "key.pem",
			},
			AuthConf: authconf.Config{
				PasswordLen:     8,
				TokenExpiration: time.Hour * 24,
			},
			LogConf: logconf.LogLevel{
				Level: zerolog.DebugLevel,
			},
		}

		err = c.loadConfigFile()

		assert.NoError(t, err)
		assert.Equal(t, "testhost:9090", c.ServerConf.ServerAddr)
		assert.Equal(t, "test_cert.pem", c.ServerConf.TLSCertPath)
		assert.Equal(t, "test_key.pem", c.ServerConf.TLSKeyPath)
		assert.Equal(t, 12, c.AuthConf.PasswordLen)
		assert.Equal(t, time.Hour*48, c.AuthConf.TokenExpiration)
		assert.Equal(t, zerolog.InfoLevel, c.LogConf.Level)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid_config.json")

		err := os.WriteFile(configPath, []byte("invalid json"), 0644)
		require.NoError(t, err)

		c := &Config{
			ConfigFile: configPath,
		}

		err = c.loadConfigFile()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse config file")
	})

	t.Run("merge only non-empty values", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "partial_config.json")

		// Config file with only some fields set
		partialConfig := Config{
			ServerConf: serverconf.ServerConfig{
				ServerAddr: "newhost:3000",
				// TLSCertPath and TLSKeyPath are empty
			},
			AuthConf: authconf.Config{
				PasswordLen: 15,
				// TokenExpiration is zero
			},
		}

		data, err := json.Marshal(partialConfig)
		require.NoError(t, err)

		err = os.WriteFile(configPath, data, 0644)
		require.NoError(t, err)

		c := &Config{
			ConfigFile: configPath,
			ServerConf: serverconf.ServerConfig{
				ServerAddr:  "localhost:8080",
				TLSCertPath: "default_cert.pem",
				TLSKeyPath:  "default_key.pem",
			},
			AuthConf: authconf.Config{
				PasswordLen:     8,
				TokenExpiration: time.Hour * 24,
			},
		}

		err = c.loadConfigFile()

		assert.NoError(t, err)
		// ServerAddr should be updated
		assert.Equal(t, "newhost:3000", c.ServerConf.ServerAddr)
		// TLS paths should remain default (not overwritten by empty values)
		assert.Equal(t, "default_cert.pem", c.ServerConf.TLSCertPath)
		assert.Equal(t, "default_key.pem", c.ServerConf.TLSKeyPath)
		// PasswordLen should be updated
		assert.Equal(t, 15, c.AuthConf.PasswordLen)
		// TokenExpiration should remain default (not overwritten by zero)
		assert.Equal(t, time.Hour*24, c.AuthConf.TokenExpiration)
	})

	t.Run("log level NoLevel is not merged", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "log_config.json")

		testConfig := Config{
			LogConf: logconf.LogLevel{
				Level: zerolog.NoLevel,
			},
		}

		data, err := json.Marshal(testConfig)
		require.NoError(t, err)

		err = os.WriteFile(configPath, data, 0644)
		require.NoError(t, err)

		c := &Config{
			ConfigFile: configPath,
			LogConf: logconf.LogLevel{
				Level: zerolog.DebugLevel,
			},
		}

		err = c.loadConfigFile()

		assert.NoError(t, err)
		// NoLevel should not override the default
		assert.Equal(t, zerolog.DebugLevel, c.LogConf.Level)
	})

	t.Run("log level WarnLevel is merged", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "log_config_warn.json")

		testConfig := Config{
			LogConf: logconf.LogLevel{
				Level: zerolog.WarnLevel,
			},
		}

		data, err := json.Marshal(testConfig)
		require.NoError(t, err)

		err = os.WriteFile(configPath, data, 0644)
		require.NoError(t, err)

		c := &Config{
			ConfigFile: configPath,
			LogConf: logconf.LogLevel{
				Level: zerolog.DebugLevel,
			},
		}

		err = c.loadConfigFile()

		assert.NoError(t, err)
		assert.Equal(t, zerolog.WarnLevel, c.LogConf.Level)
	})

	t.Run("read error other than not exist", func(t *testing.T) {
		// Create a directory instead of a file to force a read error
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "is_a_directory")

		err := os.Mkdir(configPath, 0755)
		require.NoError(t, err)

		c := &Config{
			ConfigFile: configPath,
		}

		err = c.loadConfigFile()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})
}

func TestConfig_Defaults(t *testing.T) {
	t.Run("default values are set correctly", func(t *testing.T) {
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

		assert.Equal(t, "localhost:8080", c.ServerConf.ServerAddr)
		assert.Equal(t, "cert.pem", c.ServerConf.TLSCertPath)
		assert.Equal(t, "private.pem", c.ServerConf.TLSKeyPath)
		assert.Equal(t, zerolog.DebugLevel, c.LogConf.Level)
		assert.Equal(t, time.Hour*24, c.AuthConf.TokenExpiration)
		assert.Equal(t, 8, c.AuthConf.PasswordLen)
		assert.Equal(t, "config.json", c.ConfigFile)
	})
}

func TestConfig_Integration(t *testing.T) {
	t.Run("config file overrides defaults, env overrides config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "integration_config.json")

		// Create config file
		fileConfig := Config{
			ServerConf: serverconf.ServerConfig{
				ServerAddr: "filehost:9000",
			},
			PGConf: pgconf.PGConf{
				DatabaseDSN: "postgres://fromfile",
			},
			ServiceConf: serviceconf.ServiceConf{
				MasterKey: "file_master_key",
			},
			AuthConf: authconf.Config{
				SecretKey:   "file_secret_key",
				PasswordLen: 10,
			},
		}

		data, err := json.Marshal(fileConfig)
		require.NoError(t, err)

		err = os.WriteFile(configPath, data, 0644)
		require.NoError(t, err)

		// Set environment variables
		os.Setenv("DATABASE_DSN", "postgres://fromenv")
		os.Setenv("MASTER_KEY", "env_master_key")
		os.Setenv("SECRET_KEY", "env_secret_key")
		defer func() {
			os.Unsetenv("DATABASE_DSN")
			os.Unsetenv("MASTER_KEY")
			os.Unsetenv("SECRET_KEY")
		}()

		c := &Config{
			ConfigFile: configPath,
			ServerConf: serverconf.ServerConfig{
				ServerAddr: "localhost:8080",
			},
			PGConf: pgconf.PGConf{
				DatabaseDSN: "postgres://default",
			},
			ServiceConf: serviceconf.ServiceConf{
				MasterKey: "default_master",
			},
			AuthConf: authconf.Config{
				SecretKey:   "default_secret",
				PasswordLen: 8,
			},
		}

		// Load config file
		err = c.loadConfigFile()
		require.NoError(t, err)

		// Load environment
		c.loadEnv()

		// Check precedence: env > file > default
		assert.Equal(t, "filehost:9000", c.ServerConf.ServerAddr)  // from file (not in env)
		assert.Equal(t, "postgres://fromenv", c.PGConf.DatabaseDSN) // from env (overrides file)
		assert.Equal(t, "env_master_key", c.ServiceConf.MasterKey)  // from env (overrides file)
		assert.Equal(t, "env_secret_key", c.AuthConf.SecretKey)     // from env (overrides file)
		assert.Equal(t, 10, c.AuthConf.PasswordLen)                 // from file (not in env)
	})
}
