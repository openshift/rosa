package log

import (
	"fmt"
	"regexp"

	logging "github.com/sirupsen/logrus"
)

func GetLogger() *Log {
	// Create the logger
	logger := logging.New()
	// Set logger level for your debug command
	logger.SetLevel(logging.InfoLevel)
	return &Log{logger: logger, redActSensitive: true}
}

var Logger *Log = GetLogger()

type Log struct {
	logger          *logging.Logger
	redActSensitive bool
}

func (l *Log) NeedRedact(originalString string, regexP *regexp.Regexp) bool {
	return regexP.MatchString(originalString)
}
func (l *Log) Redact(fmtedString string) string {
	if !l.redActSensitive {
		return fmtedString
	}
	for _, regexP := range RedactKeyList {
		if l.NeedRedact(fmtedString, regexP) {
			l.logger.Debugf("Got need redacted string from log match regex %s", regexP.String())
			fmtedString = regexP.ReplaceAllString(fmtedString, fmt.Sprintf(`$1"%s$3`, RedactValue))
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
	fmtString = l.Redact(fmtString)
	l.logger.Info(fmtString)
}

func (l *Log) Errorf(fmtString string, args ...interface{}) {
	if len(args) != 0 {
		fmtString = fmt.Sprintf(fmtString, args...)
	}
	l.Error(fmtString)
}

func (l *Log) Error(fmtString string) {
	fmtString = l.Redact(fmtString)
	l.logger.Error(fmtString)
}

func (l *Log) Warnf(fmtString string, args ...interface{}) {
	if len(args) != 0 {
		fmtString = fmt.Sprintf(fmtString, args...)
	}
	l.Warn(fmtString)
}

func (l *Log) Warn(fmtString string) {
	fmtString = l.Redact(fmtString)
	l.logger.Warn(fmtString)
}

func (l *Log) Debugf(fmtString string, args ...interface{}) {
	if len(args) != 0 {
		fmtString = fmt.Sprintf(fmtString, args...)
	}
	l.Debug(fmtString)
}

func (l *Log) Debug(fmtString string) {
	fmtString = l.Redact(fmtString)
	l.logger.Debug(fmtString)
}
