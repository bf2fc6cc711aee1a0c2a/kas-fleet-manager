package db

import (
	"context"
	"strings"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"gorm.io/gorm/logger"
)

type customLoggerWithMetricsCollector struct {
}

// LogMode sets the log level
func (l customLoggerWithMetricsCollector) LogMode(level logger.LogLevel) logger.Interface {
	return logger.Default.LogMode(level)
}

// Info print information
func (l customLoggerWithMetricsCollector) Info(ctx context.Context, msg string, data ...interface{}) {
	logger.Default.Info(ctx, msg, data...)
}

// Warn print warning message
func (l customLoggerWithMetricsCollector) Warn(ctx context.Context, msg string, data ...interface{}) {
	logger.Default.Warn(ctx, msg, data...)
}

// Error print error message
func (l customLoggerWithMetricsCollector) Error(ctx context.Context, msg string, data ...interface{}) {
	logger.Default.Error(ctx, msg, data...)
}

// Trace trace the sql query and its execution time.
func (l customLoggerWithMetricsCollector) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	status := "success"

	if err != nil {
		status = "failure"
	}

	sql, _ := fc()
	sql = strings.TrimLeft(sql, " ")
	tokens := strings.Split(sql, " ")
	metrics.IncreaseDatabaseQueryCount(status, tokens[0])
	metrics.UpdateDatabaseQueryDurationMetric(status, tokens[0], elapsed)
}
