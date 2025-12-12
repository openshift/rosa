package logforwarders

import (
	"fmt"

	"github.com/openshift/rosa/pkg/reporter"
)

type DescribeLogForwarderUserOptions struct {
	logForwarder string
}

type DescribeLogForwarderOptions struct {
	reporter reporter.Logger
	args     DescribeLogForwarderUserOptions
}

func NewDescribeLogForwarderUserOptions() DescribeLogForwarderUserOptions {
	return DescribeLogForwarderUserOptions{logForwarder: ""}
}

func NewDescribeLogForwarderOptions() *DescribeLogForwarderOptions {
	return &DescribeLogForwarderOptions{
		reporter: reporter.CreateReporter(),
		args:     NewDescribeLogForwarderUserOptions(),
	}
}

func (i *DescribeLogForwarderOptions) Bind(args DescribeLogForwarderUserOptions) error {
	if args.logForwarder == "" {
		return fmt.Errorf("you need to specify a log forwarder ID")
	}
	logForwarderKey := args.logForwarder
	if !logForwarderKeyRE.MatchString(logForwarderKey) {
		return fmt.Errorf(
			"log forwarder identifier '%s' isn't valid: it must contain only lowercase letters and digits",
			logForwarderKey,
		)
	}
	i.args.logForwarder = args.logForwarder
	return nil
}
