package datastore

import (
	zLogger "github.com/mrz1836/go-logger"
	gLogger "gorm.io/gorm/logger"
)

// DatabaseLogWrapper is a special wrapper for the GORM logger
type DatabaseLogWrapper struct {
	zLogger.GormLoggerInterface
}

// LogMode will set the log level/mode
func (d *DatabaseLogWrapper) LogMode(level gLogger.LogLevel) gLogger.Interface {
	newLogger := *d
	switch level {
	case gLogger.Info:
		newLogger.GormLoggerInterface = newLogger.SetMode(zLogger.Info)
	case gLogger.Warn:
		newLogger.GormLoggerInterface = newLogger.SetMode(zLogger.Warn)
	case gLogger.Error:
		newLogger.GormLoggerInterface = newLogger.SetMode(zLogger.Error)
	case gLogger.Silent:
		newLogger.GormLoggerInterface = newLogger.SetMode(zLogger.Silent)
	}

	return &newLogger
}
