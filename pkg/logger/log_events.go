package logger

import (
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
)

func (l *logger) Infof(format string, args ...interface{}) {
	prefixed := l.prepareLogPrefix(format, args...)
	glog.V(glog.Level(l.level)).Infof(prefixed)
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

func (l *logger) Error(err error) {
	glog.Error(err)
	if l.sentryHub == nil {
		sentry.CaptureException(err)
		return
	}
	l.sentryHub.CaptureException(err)
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
		glog.Warning("Sentry hub not present in logger")
		sentry.CaptureEvent(event)
		return
	}
	l.sentryHub.CaptureEvent(event)
}
