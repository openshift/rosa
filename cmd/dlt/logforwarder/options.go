package logforwarder

import (
	"fmt"

	"github.com/openshift/rosa/pkg/reporter"
)

type DeleteLogForwarderUserOptions struct {
	logForwarder string
}

type DeleteLogForwarderOptions struct {
	reporter reporter.Logger

	args *DeleteLogForwarderUserOptions
}

func NewDeleteLogForwarderUserOptions() *DeleteLogForwarderUserOptions {
	return &DeleteLogForwarderUserOptions{logForwarder: ""}
}

func NewDeleteLogForwarderOptions() *DeleteLogForwarderOptions {
	return &DeleteLogForwarderOptions{
		reporter: reporter.CreateReporter(),
		args:     &DeleteLogForwarderUserOptions{},
	}
}

func (l *DeleteLogForwarderOptions) LogForwarder() string {
	return l.args.logForwarder
}

func (l *DeleteLogForwarderOptions) Bind(args *DeleteLogForwarderUserOptions, argv []string) error {
	l.args = args
	if l.LogForwarder() == "" {
		if len(argv) > 0 {
			l.args.logForwarder = argv[0]
		}
	}
	if args.logForwarder == "" {
		return fmt.Errorf("you must specify a log forwarder ID with '--log-forwarder'")
	}
	if !logForwarderKeyRE.MatchString(args.logForwarder) {
		return fmt.Errorf(
			"log forwarder ID '%s' is not valid: it must contain only lowercase letters and digits",
			args.logForwarder,
		)
	}
	l.args.logForwarder = args.logForwarder
	return nil
}
