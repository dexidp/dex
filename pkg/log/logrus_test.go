package log

import "testing"

func TestLogrusLoggerImplementsLoggerInterface(t *testing.T) {
	var i interface{} = new(LogrusLogger)
	if _, ok := i.(Logger); !ok {
		t.Errorf("expected %T to implement Logger interface", i)
	}
}
