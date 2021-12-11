package log

func Deprecated(logger Logger, f string, args ...interface{}) {
	logger.Warnf("Deprecated: "+f, args...)
}
