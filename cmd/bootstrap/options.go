package bootstrap

import (
	bsOpts "github.com/openshift/rosa/pkg/options/bootstrap"
	"github.com/openshift/rosa/pkg/reporter"
)

type BootstrapOptions struct {
	reporter *reporter.Object

	args *bsOpts.BootstrapUserOptions
}

func NewBootstrapUserOptions() *bsOpts.BootstrapUserOptions {
	return &bsOpts.BootstrapUserOptions{
		Params: []string{},
	}
}

func NewBootstrapOptions() *BootstrapOptions {
	return &BootstrapOptions{
		reporter: reporter.CreateReporter(),
	}
}

func (m *BootstrapOptions) Bootstrap() *bsOpts.BootstrapUserOptions {
	return m.args
}
