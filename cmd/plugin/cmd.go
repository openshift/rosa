package plugin

import (
	"github.com/spf13/cobra"

	plugin "github.com/openshift/rosa/cmd/plugin/list"
)

var aliases = []string{"plugins"}

const (
	use   = "plugin"
	short = "Get information about plugins"
	long  = "Get information about installed ROSA plugins"
)

func NewPluginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   short,
		Long:    long,
		Args:    cobra.MinimumNArgs(1),
	}
	cmd.AddCommand(plugin.NewListRosaPlugins())
	return cmd
}
