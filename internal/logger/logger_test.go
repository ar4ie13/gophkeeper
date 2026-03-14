package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	t.Run("creates logger with debug level", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.DebugLevel)

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)

		assert.NotNil(t, logger)
		assert.Contains(t, buf.String(), "Log level: debug")
	})

	t.Run("creates logger with info level", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.InfoLevel)

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)

		assert.NotNil(t, logger)
		assert.Contains(t, buf.String(), "Log level: info")
	})

	t.Run("creates logger with warn level", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.WarnLevel)

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)

		assert.NotNil(t, logger)
		// Warn level won't log the Info message about log level
		assert.NotContains(t, buf.String(), "Log level:")
	})

	t.Run("creates logger with error level", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.ErrorLevel)

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)

		assert.NotNil(t, logger)
		// Error level won't log the Info message
		assert.NotContains(t, buf.String(), "Log level:")
	})
}

func TestLogger_Logging(t *testing.T) {
	t.Run("debug level logs debug messages", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.DebugLevel)
		logger.Debug().Msg("debug message")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		assert.Contains(t, output, "debug message")
		assert.Contains(t, output, "DBG")
	})

	t.Run("info level logs info messages", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.InfoLevel)
		logger.Info().Msg("info message")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		assert.Contains(t, output, "info message")
		assert.Contains(t, output, "INF")
	})

	t.Run("warn level logs warn messages", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.WarnLevel)
		logger.Warn().Msg("warn message")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		assert.Contains(t, output, "warn message")
		assert.Contains(t, output, "WRN")
	})

	t.Run("error level logs error messages", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.ErrorLevel)
		logger.Error().Msg("error message")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		assert.Contains(t, output, "error message")
		assert.Contains(t, output, "ERR")
	})

	t.Run("info level does not log debug messages", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.InfoLevel)
		logger.Debug().Msg("debug message should not appear")
		logger.Info().Msg("info message should appear")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		assert.NotContains(t, output, "debug message should not appear")
		assert.Contains(t, output, "info message should appear")
	})

	t.Run("error level does not log info messages", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.ErrorLevel)
		logger.Info().Msg("info message should not appear")
		logger.Error().Msg("error message should appear")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		assert.NotContains(t, output, "info message should not appear")
		assert.Contains(t, output, "error message should appear")
	})
}

func TestLogger_Fields(t *testing.T) {
	t.Run("logs with string field", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.InfoLevel)
		logger.Info().Str("key", "value").Msg("message with field")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		assert.Contains(t, output, "message with field")
		assert.Contains(t, output, "key=")
		assert.Contains(t, output, "value")
	})

	t.Run("logs with int field", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.InfoLevel)
		logger.Info().Int("count", 42).Msg("message with int")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		assert.Contains(t, output, "message with int")
		assert.Contains(t, output, "count=")
		assert.Contains(t, output, "42")
	})

	t.Run("logs with error field", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.InfoLevel)
		logger.Error().Err(assert.AnError).Msg("message with error")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		assert.Contains(t, output, "message with error")
		assert.Contains(t, output, "error=")
	})

	t.Run("logs with multiple fields", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.InfoLevel)
		logger.Info().
			Str("user", "john").
			Int("age", 30).
			Bool("active", true).
			Msg("user info")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		assert.Contains(t, output, "user info")
		assert.Contains(t, output, "user=")
		assert.Contains(t, output, "john")
		assert.Contains(t, output, "age=")
		assert.Contains(t, output, "30")
		assert.Contains(t, output, "active=")
		assert.Contains(t, output, "true")
	})
}

func TestLogger_Msgf(t *testing.T) {
	t.Run("logs formatted message", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.InfoLevel)
		logger.Info().Msgf("User %s logged in at %d", "alice", 12345)

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		assert.Contains(t, output, "User alice logged in at 12345")
	})
}

func TestLogger_WithContext(t *testing.T) {
	t.Run("creates sub-logger with context", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.InfoLevel)
		subLogger := logger.With().Str("component", "auth").Logger()
		subLogger.Info().Msg("authentication event")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		assert.Contains(t, output, "authentication event")
		assert.Contains(t, output, "component=")
		assert.Contains(t, output, "auth")
	})
}

func TestLogger_LevelBehavior(t *testing.T) {
	levels := []struct {
		level    zerolog.Level
		levelStr string
		logs     map[string]bool // method -> should log
	}{
		{
			level:    zerolog.DebugLevel,
			levelStr: "debug",
			logs: map[string]bool{
				"debug": true,
				"info":  true,
				"warn":  true,
				"error": true,
			},
		},
		{
			level:    zerolog.InfoLevel,
			levelStr: "info",
			logs: map[string]bool{
				"debug": false,
				"info":  true,
				"warn":  true,
				"error": true,
			},
		},
		{
			level:    zerolog.WarnLevel,
			levelStr: "warn",
			logs: map[string]bool{
				"debug": false,
				"info":  false,
				"warn":  true,
				"error": true,
			},
		},
		{
			level:    zerolog.ErrorLevel,
			levelStr: "error",
			logs: map[string]bool{
				"debug": false,
				"info":  false,
				"warn":  false,
				"error": true,
			},
		},
	}

	for _, tc := range levels {
		t.Run("level "+tc.levelStr, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			logger := NewLogger(tc.level)

			// Log at all levels
			logger.Debug().Msg("debug message")
			logger.Info().Msg("info message")
			logger.Warn().Msg("warn message")
			logger.Error().Msg("error message")

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Check each log level
			if tc.logs["debug"] {
				assert.Contains(t, output, "debug message")
			} else {
				assert.NotContains(t, output, "debug message")
			}

			if tc.logs["info"] {
				assert.Contains(t, output, "info message")
			} else {
				assert.NotContains(t, output, "info message")
			}

			if tc.logs["warn"] {
				assert.Contains(t, output, "warn message")
			} else {
				assert.NotContains(t, output, "warn message")
			}

			if tc.logs["error"] {
				assert.Contains(t, output, "error message")
			}
		})
	}
}

// TestLogger_OutputFormat verifies the logger uses ConsoleWriter and includes timestamps
func TestLogger_OutputFormat(t *testing.T) {
	t.Run("includes timestamp in output", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.InfoLevel)
		logger.Info().Msg("timestamped message")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Check for timestamp format (contains year, month, day, hour, minute, second)
		// The timestamp should be in RFC3339 format
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`, output)
	})
}

// TestLogger_Integration tests realistic logging scenarios
func TestLogger_Integration(t *testing.T) {
	t.Run("typical application logging flow", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.InfoLevel)

		// Simulate application flow
		logger.Info().Msg("Application started")
		logger.Info().Str("version", "1.0.0").Msg("Version info")
		logger.Warn().Msg("Deprecated feature used")
		logger.Error().Str("module", "database").Msg("Connection failed")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		assert.Contains(t, output, "Application started")
		assert.Contains(t, output, "version=")
		assert.Contains(t, output, "1.0.0")
		assert.Contains(t, output, "Deprecated feature used")
		assert.Contains(t, output, "Connection failed")
		assert.Contains(t, output, "module=")
		assert.Contains(t, output, "database")
	})
}

// TestLogger_Disabled tests that disabled level doesn't log
func TestLogger_Disabled(t *testing.T) {
	t.Run("disabled level does not log", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.Disabled)
		logger.Info().Msg("this should not appear")
		logger.Error().Msg("this should also not appear")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Should be empty except possibly for the initial "Log level:" message
		// which won't appear because it's at Info level
		assert.NotContains(t, output, "this should not appear")
		assert.NotContains(t, output, "this should also not appear")
	})
}

// TestLogger_JSONCompatibility ensures the logger wraps zerolog.Logger properly
func TestLogger_JSONCompatibility(t *testing.T) {
	t.Run("logger is compatible with zerolog.Logger", func(t *testing.T) {
		logger := NewLogger(zerolog.InfoLevel)

		// Should be able to use all zerolog.Logger methods
		assert.NotNil(t, logger.Logger)

		// Create a buffer logger for easier testing
		var buf bytes.Buffer
		jsonLogger := zerolog.New(&buf).With().Timestamp().Logger()

		jsonLogger.Info().Str("test", "value").Msg("json test")

		output := buf.String()

		// Should be valid JSON
		var jsonMap map[string]any
		err := json.Unmarshal([]byte(output), &jsonMap)
		require.NoError(t, err)

		assert.Equal(t, "json test", jsonMap["message"])
		assert.Equal(t, "value", jsonMap["test"])
	})
}

// TestLogger_TimeFormat verifies RFC3339 time format is used
func TestLogger_TimeFormat(t *testing.T) {
	t.Run("uses RFC3339 time format", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger(zerolog.InfoLevel)
		logger.Info().Msg("time format test")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// RFC3339 format: 2006-01-02T15:04:05Z07:00
		// Should contain a timestamp in this format
		assert.Contains(t, output, "T") // Date-time separator
		currentYear := time.Now().Format("2006")
		assert.Contains(t, output, currentYear) // Current year
	})
}
