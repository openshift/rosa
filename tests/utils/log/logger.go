package log

import (
	"os"
	"time"

	g "github.com/onsi/ginkgo/v2"
	"github.com/sirupsen/logrus"
)

func GetLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors:   true,
		DisableQuote:    true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
	})
	logger.SetOutput(g.GinkgoWriter)

	// Export `DEBUG=true` to enable debug level
	debug := os.Getenv("DEBUG")
	if debug == "true" {
		logger.SetLevel(logrus.DebugLevel)
	}

	sensitiveHook := &SensitiveHook{
		regexes: RedactKeyList,
	}
	logger.AddHook(sensitiveHook)

	return logger
}

var Logger *logrus.Logger = GetLogger()
