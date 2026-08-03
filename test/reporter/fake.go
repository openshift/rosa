package reportertest

import "github.com/openshift/rosa/pkg/reporter"

var _ reporter.Logger = &FakeLogger{}

// FakeLogger is a test double for reporter.Logger. Set the optional callback
// fields to capture or assert on specific log calls; nil callbacks are no-ops.
type FakeLogger struct {
	DebugFn  func(string, ...any)
	InfoFn   func(string, ...any)
	WarnFn   func(string, ...any)
	ErrorFn  func(string, ...any) error
	Terminal bool
}

func (f *FakeLogger) Debugf(format string, a ...any) {
	if f.DebugFn != nil {
		f.DebugFn(format, a...)
	}
}
func (f *FakeLogger) Infof(format string, a ...any) {
	if f.InfoFn != nil {
		f.InfoFn(format, a...)
	}
}
func (f *FakeLogger) Warnf(format string, a ...any) {
	if f.WarnFn != nil {
		f.WarnFn(format, a...)
	}
}
func (f *FakeLogger) Errorf(format string, a ...any) error {
	if f.ErrorFn != nil {
		return f.ErrorFn(format, a...)
	}
	return nil
}
func (f *FakeLogger) IsTerminal() bool { return f.Terminal }
