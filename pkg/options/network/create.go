package network

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/reporter"
)

const (
	use   = "network"
	short = "Network aws cloudformation stack"
	long  = `Network aws cloudformation stack using predefined yaml templates.
You can modify the OCM_TEMPLATE_DIR environment variable to point to the location of the cloudformation templates.`
	example = `  # Create a aws cloudformation stack
  rosa create network <template-name> --param Param1=Value1 --param Param2=Value2 `
	defaultTemplateDir = "cmd/create/network/templates"
)

type NetworkUserOptions struct {
	Params      []string
	TemplateDir string
}

type NetworkOptions struct {
	reporter *reporter.Object
	args     *NetworkUserOptions
}

func NewNetworkUserOptions() *NetworkUserOptions {
	options := &NetworkUserOptions{}

	// Set template directory from environment variable or use default
	templateDir := os.Getenv("OCM_TEMPLATE_DIR")
	if helper.HandleEscapedEmptyString(templateDir) != "" {
		options.TemplateDir = templateDir
	} else {
		options.TemplateDir = defaultTemplateDir
	}

	return options
}

func NewNetworkOptions() *NetworkOptions {
	return &NetworkOptions{
		reporter: reporter.CreateReporter(),
		args:     NewNetworkUserOptions(),
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
	flags.StringVar(&options.TemplateDir, "template-dir", defaultTemplateDir, "Use a specific template directory,"+
		" overriding the OCM_TEMPLATE_DIR environment variable.")
	flags.StringArrayVar(&options.Params, "param", []string{}, "List of parameters")

	return cmd, options
}
