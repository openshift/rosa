package Bootstrap

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/reporter"
)

const (
	use     = "bootstrap"
	short   = "Bootstrap aws cloudformation stack"
	long    = "Bootstrap aws cloudformation stack using predefined yaml templates. "
	example = `  # Create a aws cloudformation stack
  rosa bootstrap <template-name> --param Param1=Value1 --param Param2=Value2 `
)

type BootstrapUserOptions struct {
	Params []string
}

type BootstrapOptions struct {
	reporter *reporter.Object
	args     *BootstrapUserOptions
}

func NewBootstrapUserOptions() *BootstrapUserOptions {
	return &BootstrapUserOptions{}
}

func NewBootstrapOptions() *BootstrapOptions {
	return &BootstrapOptions{
		reporter: reporter.CreateReporter(),
		args:     &BootstrapUserOptions{},
	}
}

func (m *BootstrapOptions) Bootstrap() *BootstrapUserOptions {
	return m.args
}

func BuildBootstrapCommandWithOptions() (*cobra.Command, *BootstrapUserOptions) {
	options := NewBootstrapUserOptions()
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Aliases: []string{"bootstraps", "boot-strap", "boot-straps"},
		Example: example,
		Args:    cobra.ExactArgs(1),
		Hidden:  true,
	}

	flags := cmd.Flags()

	flags.StringArrayVar(&options.Params, "param", []string{}, "List of parameters")

	return cmd, options
}
