package log

import (
	"os"

	stackdriver "github.com/TV4/logrus-stackdriver-formatter"
	"github.com/sirupsen/logrus"
)

type stackdriverLogger struct {
	logger *logrus.Logger
}

type StackdriverLoggerOptions struct {
	LogLevel    string
	ServiceName string
	Version     string
}

func NewStackdriverLogger(opts *StackdriverLoggerOptions) Logger {
	formatter := stackdriver.NewFormatter(stackdriver.WithService(opts.ServiceName),
		stackdriver.WithVersion(opts.Version))

	var level logrus.Level
	err := level.UnmarshalText([]byte(opts.LogLevel))
	if err != nil {
		level = logrus.InfoLevel
	}

	return &stackdriverLogger{
		logger: &logrus.Logger{
			Out:          os.Stderr,
			Formatter:    formatter,
			Hooks:        make(logrus.LevelHooks),
			Level:        level,
			ExitFunc:     os.Exit,
			ReportCaller: false,
		},
	}
}

func (l *stackdriverLogger) Log(keyvals ...interface{}) error {
	pairs := len(keyvals)
	logger := l.logger.WithFields(logrus.Fields{})
	level := 0
	for n := 0; n < pairs-(pairs%2); n += 2 {
		key := keyvals[n].(string)
		val := keyvals[n+1]
		if key == "level" {
			level = val.(int)
		} else {
			logger = logger.WithField(key, val)
		}
	}

	// 0 - error, 1 - info,
	// 2 - debug, 3 - trace.
	switch level {
	case 0:
		logger.Error()
	case 1:
		logger.Info()
	case 2:
		logger.Debug()
	case 3:
		logger.Trace()
	default:
		logger.Info()
	}

	return nil
}
