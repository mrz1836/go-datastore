package datastore

import (
	"testing"

	zLogger "github.com/mrz1836/go-logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gLogger "gorm.io/gorm/logger"
)

func TestDatabaseLogWrapperLogMode(t *testing.T) {
	base := &DatabaseLogWrapper{GormLoggerInterface: zLogger.NewGormLogger(false, 1)}

	tests := []struct {
		name     string
		level    gLogger.LogLevel
		expected zLogger.GormLogLevel
	}{
		{
			name:     "info level",
			level:    gLogger.Info,
			expected: zLogger.Info,
		},
		{
			name:     "warn level",
			level:    gLogger.Warn,
			expected: zLogger.Warn,
		},
		{
			name:     "error level",
			level:    gLogger.Error,
			expected: zLogger.Error,
		},
		{
			name:     "silent level",
			level:    gLogger.Silent,
			expected: zLogger.Silent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper := &DatabaseLogWrapper{GormLoggerInterface: base.GormLoggerInterface}

			result := wrapper.LogMode(tt.level)
			require.IsType(t, &DatabaseLogWrapper{}, result)

			logger := result.(*DatabaseLogWrapper)
			require.NotNil(t, logger)
			assert.Equal(t, tt.expected, logger.GetMode())
		})
	}
}
