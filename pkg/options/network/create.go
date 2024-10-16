package Network

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/reporter"
)

const (
	use     = "network"
	short   = "Network AWS cloudformation stack"
	long    = "Network AWS cloudformation stack using predefined yaml templates. "
	example = `  # Create a AWS cloudformation stack
  rosa create network <template-name> --param Param1=Value1 --param Param2=Value2 ` +
		"\n\n" + `  # ROSA quick start HCP VPC example` +
		"\n" + `  rosa create network rosa-quickstart-default-vpc --param Region=us-west-2` +
		` --param Name=quickstart-stack --param AvailabilityZoneCount=1 --param VpcCidr=10.0.0.0/16` +
		"\n\n" + `  # To delete the AWS cloudformation stack` +
		"\n" + `  aws cloudformation delete-stack --stack-name <name> --region <region>`
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
		Args:    cobra.MaximumNArgs(1),
		Hidden:  true,
	}

	flags := cmd.Flags()

	flags.StringArrayVar(&options.Params, "param", []string{}, "List of parameters")

	return cmd, options
}
