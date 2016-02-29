package drivers

import "log"

type driverLogger struct {
	logger *log.Logger
}

func newDriverLogger(l *log.Logger) driverLogger {
	return driverLogger{logger: l}
}

// Log logs the parameters to the preconfigured logger.
func (l driverLogger) Log(args ...interface{}) {
	l.logger.Println(args...)
}
