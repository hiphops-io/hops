package nats

type Logger interface {
	// Log a debug statement
	Debugf(format string, v ...interface{})

	// Log an error with exact error
	Errf(err error, format string, v ...interface{})

	// Log an error
	Errorf(format string, v ...interface{})

	// Log a fatal error
	Fatalf(format string, v ...interface{})

	// Log an info statement
	Infof(format string, v ...interface{})

	// Log a notice statement
	Noticef(format string, v ...interface{})

	// Log a trace statement
	Tracef(format string, v ...interface{})

	// Log a warning statement
	Warnf(format string, v ...interface{})
}
