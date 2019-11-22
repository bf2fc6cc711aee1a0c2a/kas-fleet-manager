package logger

import (
	"context"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
)

type UHCLogger interface {
	V(level int32) UHCLogger
	Infof(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

var _ UHCLogger = &logger{}

type logger struct {
	context   context.Context
	level     int32
	accountID string
	// TODO username is unused, should we be logging it? Could be pii
	username  string
	sentryHub *sentry.Hub
}

// NewUHCLogger creates a new logger instance with a default verbosity of 1
func NewUHCLogger(ctx context.Context) UHCLogger {
	logger := &logger{
		context:   ctx,
		level:     1,
		username:  getUsernameFromContext(ctx),
		sentryHub: sentry.GetHubFromContext(ctx),
	}
	return logger
}

func (l *logger) prepareLogPrefix(format string, args ...interface{}) string {
	orig := fmt.Sprintf(format, args...)
	prefix := " "

	if txid, ok := l.context.Value("txid").(int64); ok {
		prefix = fmt.Sprintf("[tx_id=%d]%s", txid, prefix)
	}

	if l.accountID != "" {
		prefix = fmt.Sprintf("[accountID=%s]%s", l.accountID, prefix)
	}

	if opid, ok := l.context.Value(OpIDKey).(string); ok {
		prefix = fmt.Sprintf("[opid=%s]%s", opid, prefix)
	}

	return fmt.Sprintf("%s%s", prefix, orig)
}

func (l *logger) V(level int32) UHCLogger {
	return &logger{
		context:   l.context,
		accountID: l.accountID,
		username:  l.username,
		level:     level,
	}
}

func (l *logger) Infof(format string, args ...interface{}) {
	prefixed := l.prepareLogPrefix(format, args...)
	glog.V(glog.Level(l.level)).Infof(prefixed)
	// TODO do we really _want_ info-level reporting to sentry?
	// l.captureSentryEvent(sentry.LevelInfo, format, args...)
}

func (l *logger) Warningf(format string, args ...interface{}) {
	prefixed := l.prepareLogPrefix(format, args...)
	glog.Warningln(prefixed)
	l.captureSentryEvent(sentry.LevelWarning, format, args...)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	prefixed := l.prepareLogPrefix(format, args...)
	glog.Errorln(prefixed)
	l.captureSentryEvent(sentry.LevelError, format, args...)
}

func (l *logger) Fatalf(format string, args ...interface{}) {
	prefixed := l.prepareLogPrefix(format, args...)
	glog.Fatalln(prefixed)
	l.captureSentryEvent(sentry.LevelFatal, format, args...)
}

func (l *logger) captureSentryEvent(level sentry.Level, format string, args ...interface{}) {
	event := sentry.NewEvent()
	event.Level = level
	event.Message = fmt.Sprintf(format, args...)
	if l.sentryHub == nil {
		sentry.CaptureException(fmt.Errorf("Sentry hub not present in logger"))
		sentry.CaptureEvent(event)
		return
	}
	l.sentryHub.CaptureEvent(event)
}

func getUsernameFromContext(ctx context.Context) string {
	username := ctx.Value("username")
	if username == nil {
		return ""
	}
	return fmt.Sprintf("%v", username)
}
