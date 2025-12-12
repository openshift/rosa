package logforwarder

import (
	"github.com/openshift/rosa/pkg/reporter"
)

type CreateLogForwarderUserOptions struct {
	logFwdConfig string
}

type CreateLogForwarderOptions struct {
	reporter reporter.Logger
	args     *CreateLogForwarderUserOptions
}

func NewCreateLogForwarderUserOptions() *CreateLogForwarderUserOptions {
	return &CreateLogForwarderUserOptions{logFwdConfig: ""}
}

func NewCreateLogForwarderOptions() *CreateLogForwarderOptions {
	return &CreateLogForwarderOptions{
		reporter: reporter.CreateReporter(),
		args:     NewCreateLogForwarderUserOptions(),
	}
}

func (i *CreateLogForwarderOptions) Bind(args *CreateLogForwarderUserOptions) error {
	i.args.logFwdConfig = args.logFwdConfig
	return nil
}
