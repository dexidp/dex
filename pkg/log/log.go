package log

import (
	"fmt"
	"io"
	"log"
	"os"
)

const (
	calldepth = 2
)

type Level int

const (
	// Log level in order of precidence from lowest to highest.
	LevelDebug   Level = iota
	LevelInfo    Level = iota
	LevelWarning Level = iota
	LevelError   Level = iota
	LevelFatal   Level = iota
	LevelOff     Level = iota // do not print any logs
)

var levelStr = map[Level]string{
	LevelDebug:   "DEBUG",
	LevelInfo:    "INFO",
	LevelWarning: "WARN",
	LevelError:   "ERROR",
	LevelFatal:   "FATAL",
}

var (
	logger   = log.New(os.Stderr, "", 0)
	minLevel = LevelInfo
)

func output(lvl Level, v ...interface{}) {
	if minLevel > lvl {
		return
	}
	logger.Output(calldepth, levelStr[lvl]+" "+fmt.Sprint(v...))
}

func outputf(lvl Level, format string, v ...interface{}) {
	if minLevel > lvl {
		return
	}
	logger.Output(calldepth, levelStr[lvl]+" "+fmt.Sprintf(format, v...))
}

func EnableTimestamps() {
	logger.SetFlags(logger.Flags() | log.Ldate | log.Ltime)
}

func EnableDebug() {
	SetLevel(LevelDebug)
}

// SetLevel sets the minimum log precidence. Log statements below this
// precidence will not be displayed.
func SetLevel(lvl Level) {
	minLevel = lvl
}

func Debug(v ...interface{}) { output(LevelDebug, v...) }

func Debugf(format string, v ...interface{}) {
	outputf(LevelDebug, format, v...)
}

func Info(v ...interface{}) {
	output(LevelInfo, v...)
}

func Infof(format string, v ...interface{}) {
	outputf(LevelInfo, format, v...)
}

func Error(v ...interface{}) {
	output(LevelError, v...)
}

func Errorf(format string, v ...interface{}) {
	outputf(LevelError, format, v...)
}

func Warning(v ...interface{}) {
	output(LevelWarning, v...)
}

func Warningf(format string, v ...interface{}) {
	outputf(LevelWarning, format, v...)
}

func Fatal(v ...interface{}) {
	output(LevelFatal, v...)
	os.Exit(1)
}

func Fatalf(format string, v ...interface{}) {
	outputf(LevelFatal, format, v...)
	os.Exit(1)
}

type logWriter Level

func (l logWriter) Write(p []byte) (n int, err error) {
	output(Level(l), string(p))
	return len(p), nil
}

func InfoWriter() io.Writer {
	return logWriter(LevelInfo)
}
