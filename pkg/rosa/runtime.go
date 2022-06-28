package rosa

import (
	"os"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/sirupsen/logrus"
)

type Runtime struct {
	Reporter    *reporter.Object
	Logger      *logrus.Logger
	OCMClient   *ocm.Client
	AWSClient   aws.Client
	Creator     *aws.Creator
	FlagChecker *arguments.FlagCheck
}

func NewRuntime() *Runtime {
	reporter := reporter.CreateReporterOrExit()
	logger := logging.NewLogger()
	return &Runtime{Reporter: reporter, Logger: logger}
}

// Adds an OCM client to the runtime. Requires a deferred call to `.Cleanup()` to close connections.
func (r *Runtime) WithOCM() *Runtime {
	if r.OCMClient == nil {
		r.OCMClient = ocm.CreateNewClientOrExit(r.Logger, r.Reporter)
	}
	return r
}

// Adds an AWS client to the runtime
func (r *Runtime) WithAWS() *Runtime {
	if r.AWSClient == nil {
		r.AWSClient = aws.CreateNewClientOrExit(r.Logger, r.Reporter)
	}
	if r.Creator == nil {
		var err error
		r.Creator, err = r.AWSClient.GetCreator()
		if err != nil {
			r.Reporter.Errorf("Failed to get AWS creator: %v", err)
			os.Exit(1)
		}
	}
	return r
}

func (r *Runtime) WithFlagChecker() *Runtime {
	if r.FlagChecker == nil {
		r.FlagChecker = arguments.NewFlagCheck()
	}
	return r
}

func (r *Runtime) Cleanup() {
	if r.OCMClient != nil {
		if err := r.OCMClient.Close(); err != nil {
			r.Reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}
}
