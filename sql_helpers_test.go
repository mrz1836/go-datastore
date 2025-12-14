package datastore

import (
	"io"
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// TestGetDNS exercises the DSN builder to ensure it respects file paths and sharing flags.
func TestGetDNS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		path        string
		shared      bool
		expectedDSN string
	}{
		{
			name:        "default in-memory",
			expectedDSN: dsnDefault,
		},
		{
			name:        "file path without sharing",
			path:        "/tmp/sqlite.db",
			expectedDSN: "/tmp/sqlite.db",
		},
		{
			name:        "file path shared no query",
			path:        "/tmp/sqlite.db",
			shared:      true,
			expectedDSN: "/tmp/sqlite.db?cache=shared",
		},
		{
			name:        "file path with existing query",
			path:        "/tmp/sqlite.db?mode=ro",
			shared:      true,
			expectedDSN: "/tmp/sqlite.db?mode=ro&cache=shared",
		},
		{
			name:        "shared memory",
			shared:      true,
			expectedDSN: dsnDefault + "?cache=shared",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expectedDSN, getDNS(tt.path, tt.shared))
		})
	}
}

// TestContains verifies the recursive contains helper properly detects substrings.
func TestContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		subject  string
		substr   string
		expected bool
	}{
		{"prefix match", "cache=shared", "cache=shared", true},
		{"middle match", "abc-cache=shared-suffix", "cache=shared", true},
		{"no match", "something else", "cache=shared", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, contains(tt.subject, tt.substr))
		})
	}
}

// TestGetGormSessionConfig ensures logger and prepared statement flags are configured predictably.
func TestGetGormSessionConfig(t *testing.T) {
	t.Parallel()

	customLogger := glogger.New(log.New(io.Discard, "", 0), glogger.Config{LogLevel: glogger.Warn})

	tests := []struct {
		name           string
		prepared       bool
		debug          bool
		optionalLogger glogger.Interface
		expectedLevel  glogger.LogLevel
	}{
		{"default logger silent", false, false, nil, glogger.Silent},
		{"default logger debug", true, true, nil, glogger.Info},
		{"custom logger respected", false, false, customLogger, glogger.Warn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := getGormSessionConfig(tt.prepared, tt.debug, tt.optionalLogger)
			require.NotNil(t, cfg)
			assert.Equal(t, tt.prepared, cfg.PrepareStmt)

			assert.Equal(t, tt.expectedLevel, getLogLevel(t, cfg.Logger))
		})
	}
}

// TestGetGormConfig validates naming strategy and logger behavior.
func TestGetGormConfig(t *testing.T) {
	t.Parallel()

	customLogger := glogger.New(log.New(io.Discard, "", 0), glogger.Config{LogLevel: glogger.Warn})

	tests := []struct {
		name           string
		tablePrefix    string
		prepared       bool
		debug          bool
		optionalLogger glogger.Interface
		expectedPrefix string
		expectedLevel  glogger.LogLevel
	}{
		{"prefix and default logger", "pref", false, false, nil, "pref_", glogger.Silent},
		{"no prefix debug logger", "", true, true, nil, "", glogger.Info},
		{"custom logger", "custom", false, false, customLogger, "custom_", glogger.Warn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := getGormConfig(tt.tablePrefix, tt.prepared, tt.debug, tt.optionalLogger)
			require.NotNil(t, cfg)

			assert.Equal(t, tt.prepared, cfg.PrepareStmt)

			assert.Equal(t, tt.expectedLevel, getLogLevel(t, cfg.Logger))

			// Verify the naming strategy applies prefixes correctly
			assert.Equal(t, tt.expectedPrefix+"users", cfg.NamingStrategy.TableName("users"))
		})
	}
}

// TestCloseSQLDatabase covers nil handling and successful closes.
func TestCloseSQLDatabase(t *testing.T) {
	t.Parallel()

	t.Run("nil database", func(t *testing.T) {
		assert.NoError(t, closeSQLDatabase(nil))
	})

	t.Run("open sqlite database", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(dsnDefault), &gorm.Config{})
		require.NoError(t, err)

		assert.NoError(t, closeSQLDatabase(db))
	})
}

// TestSQLConfigDefaults ensures SQL defaults fill missing configuration depending on engine.
func TestSQLConfigDefaults(t *testing.T) {
	t.Parallel()

	t.Run("mysql defaults", func(t *testing.T) {
		cfg := (&SQLConfig{}).sqlDefaults(MySQL)
		assert.Equal(t, defaultMySQLPort, cfg.Port)
		assert.Equal(t, defaultMySQLHost, cfg.Host)
		assert.Equal(t, defaultTimeZone, cfg.TimeZone)
		assert.Equal(t, defaultDatabaseTxTimeout, cfg.TxTimeout)
	})

	t.Run("postgres defaults", func(t *testing.T) {
		cfg := (&SQLConfig{}).sqlDefaults(PostgreSQL)
		assert.Equal(t, defaultPostgreSQLPort, cfg.Port)
		assert.Equal(t, defaultPostgreSQLHost, cfg.Host)
		assert.Equal(t, defaultPostgreSQLSslMode, cfg.SslMode)
	})

	t.Run("existing values remain", func(t *testing.T) {
		cfg := (&SQLConfig{ // pre-set values should remain unchanged
			CommonConfig: CommonConfig{Debug: true},
			Host:         "custom",
			Port:         "9999",
			TimeZone:     "UTC",
			SslMode:      "disable",
			TxTimeout:    5 * time.Second,
		}).sqlDefaults(PostgreSQL)

		assert.Equal(t, "custom", cfg.Host)
		assert.Equal(t, "9999", cfg.Port)
		assert.Equal(t, "UTC", cfg.TimeZone)
		assert.Equal(t, "disable", cfg.SslMode)
		assert.Equal(t, 5*time.Second, cfg.TxTimeout)
	})
}

func getLogLevel(t *testing.T, logger glogger.Interface) glogger.LogLevel {
	t.Helper()

	value := reflect.ValueOf(logger)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	config := value.FieldByName("Config")
	require.True(t, config.IsValid(), "logger missing Config field")

	level := config.FieldByName("LogLevel")
	require.True(t, level.IsValid(), "logger missing LogLevel field")

	return glogger.LogLevel(level.Int())
}
