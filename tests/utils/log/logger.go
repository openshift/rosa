package log

import (
	"context"
	"fmt"
	"regexp"
	"time"

	// logging "github.com/sirupsen/logrus"
	g "github.com/onsi/ginkgo/v2"
	"github.com/openshift-online/ocm-sdk-go/logging"
)

func GetLogger() *Log {
	// Create the logger
	logger, _ := logging.
		NewStdLoggerBuilder().
		Streams(g.GinkgoWriter, g.GinkgoWriter).
		Debug(false).
		Build()
	return &Log{
		logger:          logger,
		logContext:      context.TODO(),
		format:          time.RFC3339,
		redActSensitive: true,
	}
}

var Logger *Log = GetLogger()

type Log struct {
	logger          *logging.StdLogger
	logContext      context.Context
	format          string
	redActSensitive bool
}

func (l *Log) NeedRedact(originalString string, regexP *regexp.Regexp) bool {
	return regexP.MatchString(originalString)
}

func (l *Log) DecorateLog(level string, message string) string {
	now := time.Now().Format(l.format)
	return fmt.Sprintf("%s - %s : %s", now, level, message)
}
func (l *Log) Redact(fmtedString string) string {
	if !l.redActSensitive {
		return fmtedString
	}
	for _, regexP := range RedactKeyList {
		if l.NeedRedact(fmtedString, regexP) {
			l.logger.Debug(l.logContext, "Got need redacted string from log match regex %s", regexP.String())
			fmtedString = regexP.ReplaceAllString(fmtedString, fmt.Sprintf(`$1%s$3`, RedactValue))
		}
	}
	return fmtedString
}
func (l *Log) Infof(fmtString string, args ...interface{}) {
	if len(args) != 0 {
		fmtString = fmt.Sprintf(fmtString, args...)
	}
	l.Info(fmtString)
}

func (l *Log) Info(fmtString string) {
	fmtString = l.DecorateLog(info, l.Redact(fmtString))
	l.logger.Info(l.logContext, fmtString)
}

func (l *Log) Errorf(fmtString string, args ...interface{}) {
	if len(args) != 0 {
		fmtString = fmt.Sprintf(fmtString, args...)
	}
	l.Error(fmtString)
}

func (l *Log) Error(fmtString string) {
	fmtString = l.DecorateLog(err, l.Redact(fmtString))
	l.logger.Error(l.logContext, fmtString)
}

func (l *Log) Warnf(fmtString string, args ...interface{}) {
	if len(args) != 0 {
		fmtString = fmt.Sprintf(fmtString, args...)
	}
	l.Warn(fmtString)
}

func (l *Log) Warn(fmtString string) {
	fmtString = l.DecorateLog(warn, l.Redact(fmtString))
	l.logger.Warn(l.logContext, fmtString)
}

func (l *Log) Debugf(fmtString string, args ...interface{}) {
	if len(args) != 0 {
		fmtString = fmt.Sprintf(fmtString, args...)
	}
	l.Debug(fmtString)
}

func (l *Log) Debug(fmtString string) {
	fmtString = l.DecorateLog(debug, l.Redact(fmtString))
	l.logger.Debug(l.logContext, fmtString)
}

func (l *Log) Fatal(fmtString string) {
	fmtString = l.DecorateLog(fatal, l.Redact(fmtString))
	l.logger.Fatal(l.logContext, fmtString)
}
