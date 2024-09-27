package network

import (
	bsOpts "github.com/openshift/rosa/pkg/options/network"
	"github.com/openshift/rosa/pkg/reporter"
)

type NetworkOptions struct {
	reporter *reporter.Object

	args *bsOpts.NetworkUserOptions
}

func NewNetworkUserOptions() *bsOpts.NetworkUserOptions {
	return &bsOpts.NetworkUserOptions{
		Params: []string{},
	}
}

func NewNetworkOptions() *NetworkOptions {
	return &NetworkOptions{
		reporter: reporter.CreateReporter(),
	}
}

func (m *NetworkOptions) Network() *bsOpts.NetworkUserOptions {
	return m.args
}
