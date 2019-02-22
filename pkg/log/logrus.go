package log

import "github.com/sirupsen/logrus"

// LogrusLogger is an adapter for Logrus implementing the Logger interface.
type LogrusLogger struct {
	logger logrus.FieldLogger
}

// NewLogrusLogger returns a new Logger wrapping Logrus.
func NewLogrusLogger(logger logrus.FieldLogger) *LogrusLogger {
	return &LogrusLogger{
		logger: logger,
	}
}

// Info logs an Info level event.
func (l *LogrusLogger) Info(msg string) {
	l.logger.Info(msg)
}

// Warn logs a Warn level event.
func (l *LogrusLogger) Warn(msg string) {
	l.logger.Warn(msg)
}

// Debugf formats and logs a Debug level event.
func (l *LogrusLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

// Infof formats and logs an Info level event.
func (l *LogrusLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

// Errorf formats and logs n Error level event.
func (l *LogrusLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}
