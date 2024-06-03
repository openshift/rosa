package log

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/sirupsen/logrus"
)

// Inspired from https://github.com/whuang8/redactrus/blob/master/redactrus.go

// SensitiveHook is a logrus hook for redacting information from logs via regexp
type SensitiveHook struct {
	// List of regex to match. They should be returning 3 groups, the second one
	// being the one which contains the sensitive data and which will be
	// redacted
	regexes []*regexp.Regexp
}

// All logrus levels are returned
func (h *SensitiveHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// LevelThreshold returns a []logrus.Level including all levels
// above and including the level given. If the provided level does not exit,
// an empty slice is returned.
func LevelThreshold(l logrus.Level) []logrus.Level {
	if int(l) > len(logrus.AllLevels) {
		return []logrus.Level{}
	}
	return logrus.AllLevels[:l+1]
}

// Fire redacts values in an log Entry that match the regex
func (h *SensitiveHook) Fire(e *logrus.Entry) error {
	for _, regexP := range h.regexes {
		// Redact based on key matching in Data fields
		for k, v := range e.Data {
			// Logrus Field values can be nil
			if v == nil {
				continue
			}

			// Redact based on value matching in Data fields
			switch reflect.TypeOf(v).Kind() {
			case reflect.String:
				switch vv := v.(type) {
				case string:
					e.Data[k] = regexP.ReplaceAllString(vv, fmt.Sprintf(`$1%s$3`, RedactValue))
				default:
					e.Data[k] = regexP.ReplaceAllString(fmt.Sprint(v), fmt.Sprintf(`$1%s$3`, RedactValue))
				}
				continue
			// prevent nil *fmt.Stringer from reaching handler below
			case reflect.Ptr:
				if reflect.ValueOf(v).IsNil() {
					continue
				}
			}

			// Handle fmt.Stringer type.
			if vv, ok := v.(fmt.Stringer); ok {
				e.Data[k] = regexP.ReplaceAllString(vv.String(), fmt.Sprintf(`$1%s$3`, RedactValue))
				continue
			}

		}

		// Redact based on text matching in the Message field
		e.Message = regexP.ReplaceAllString(e.Message, fmt.Sprintf(`$1%s$3`, RedactValue))
	}

	return nil
}
