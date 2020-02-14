package log

import "testing"

func TestStackdriverLogger_Log(t *testing.T) {
	logger := NewStackdriverLogger(&StackdriverLoggerOptions{
		LogLevel: "trace",
	})
	logger.Log("level", 1,
		"component", "server",
		"action", "start",
		"hello", "what?",
		"hmm")
}
