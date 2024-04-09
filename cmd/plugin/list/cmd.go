package plugin

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/plugin"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "list"
	short   = "List ROSA plugins"
	long    = "List all the plugins available in the users executable path"
	example = "rosa plugins list"
)

func NewListRosaPlugins() *cobra.Command {
	return &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Example: example,
		Args:    cobra.NoArgs,
		Run:     rosa.DefaultRunner(rosa.DefaultRuntime(), ListPluginRunner()),
	}
}

func ListPluginRunner() rosa.CommandRunner {
	return func(_ context.Context, runtime *rosa.Runtime, command *cobra.Command, args []string) error {
		pluginHandler := plugin.NewDefaultPluginHandler()
		plugins, err := pluginHandler.FindPlugins()
		if err != nil {
			return err
		}

		if len(plugins) > 0 {
			for _, cr := range plugins {
				fmt.Printf(""+
					"- Name:  %s\n"+
					"  Path:  %s\n",
					cr.Name,
					cr.Path,
				)
			}
		}
		return nil
	}
}
