package Network

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/reporter"
)

const (
	use     = "network"
	short   = "Network aws cloudformation stack"
	long    = "Network aws cloudformation stack using predefined yaml templates. "
	example = `  # Create a aws cloudformation stack
  rosa network <template-name> --param Param1=Value1 --param Param2=Value2 `
)

type NetworkUserOptions struct {
	Params []string
}

type NetworkOptions struct {
	reporter *reporter.Object
	args     *NetworkUserOptions
}

func NewNetworkUserOptions() *NetworkUserOptions {
	return &NetworkUserOptions{}
}

func NewNetworkOptions() *NetworkOptions {
	return &NetworkOptions{
		reporter: reporter.CreateReporter(),
		args:     &NetworkUserOptions{},
	}
}

func (m *NetworkOptions) Network() *NetworkUserOptions {
	return m.args
}

func BuildNetworkCommandWithOptions() (*cobra.Command, *NetworkUserOptions) {
	options := NewNetworkUserOptions()
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Aliases: []string{"networks"},
		Example: example,
		Args:    cobra.ExactArgs(1),
		Hidden:  true,
	}

	flags := cmd.Flags()

	flags.StringArrayVar(&options.Params, "param", []string{}, "List of parameters")

	return cmd, options
}
