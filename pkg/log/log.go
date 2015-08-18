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

var (
	logger = log.New(os.Stderr, "", 0)
	debug  = false
)

func EnableTimestamps() {
	logger.SetFlags(logger.Flags() | log.Ldate | log.Ltime)
}

func EnableDebug() {
	debug = true
}

func Debug(v ...interface{}) {
	if debug {
		logger.Output(calldepth, header("DEBUG", fmt.Sprint(v...)))
	}
}

func Debugf(format string, v ...interface{}) {
	if debug {
		logger.Output(calldepth, header("DEBUG", fmt.Sprintf(format, v...)))
	}
}

func Info(v ...interface{}) {
	logger.Output(calldepth, header("INFO", fmt.Sprint(v...)))
}

func Infof(format string, v ...interface{}) {
	logger.Output(calldepth, header("INFO", fmt.Sprintf(format, v...)))
}

func Error(v ...interface{}) {
	logger.Output(calldepth, header("ERROR", fmt.Sprint(v...)))
}

func Errorf(format string, v ...interface{}) {
	logger.Output(calldepth, header("ERROR", fmt.Sprintf(format, v...)))
}

func Warning(v ...interface{}) {
	logger.Output(calldepth, header("WARN", fmt.Sprint(v...)))
}

func Warningf(format string, v ...interface{}) {
	logger.Output(calldepth, header("WARN", fmt.Sprintf(format, v...)))
}

func Fatal(v ...interface{}) {
	logger.Output(calldepth, header("FATAL", fmt.Sprint(v...)))
	os.Exit(1)
}

func Fatalf(format string, v ...interface{}) {
	logger.Output(calldepth, header("FATAL", fmt.Sprintf(format, v...)))
	os.Exit(1)
}

func header(lvl, msg string) string {
	return fmt.Sprintf("%s: %s", lvl, msg)
}

type logWriter string

func (l logWriter) Write(p []byte) (n int, err error) {
	logger.Output(calldepth, header(string(l), string(p)))
	return len(p), nil
}

func InfoWriter() io.Writer {
	return logWriter("INFO")
}
