package driver

import (
	"log"

	"github.com/aws/smithy-go/logging"
)

type driverLogger struct {
	logger *log.Logger
}

func newDriverLogger(l *log.Logger) driverLogger {
	return driverLogger{logger: l}
}

// Logf implements the aws.Logger interface for the v2 SDK.
func (l driverLogger) Logf(classification logging.Classification, format string, v ...interface{}) {
	l.logger.Printf(format, v...)
}
