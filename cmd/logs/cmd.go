/*
Copyright (c) 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package logs

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/logs/install"
	"github.com/openshift/rosa/cmd/logs/uninstall"
	"github.com/openshift/rosa/pkg/arguments"
)

var Cmd = &cobra.Command{
	Use:     "logs",
	Aliases: []string{"log"},
	Short:   "Show installation or uninstallation logs for a cluster",
	Long:    "Show installation or uninstallation logs for a cluster",
	Example: `  # Show install logs for a cluster named 'mycluster'
  rosa logs install --cluster=mycluster

  # Show uninstall logs for a cluster named 'mycluster'
  rosa logs uninstall --cluster=mycluster`,
	Args: cobra.NoArgs,
}

func init() {
	Cmd.AddCommand(install.Cmd)
	Cmd.AddCommand(uninstall.Cmd)

	flags := Cmd.PersistentFlags()
	arguments.AddProfileFlag(flags)
	arguments.AddRegionFlag(flags)
	globallyAvailableCommands := []*cobra.Command{install.Cmd, uninstall.Cmd}
	arguments.MarkRegionDeprecated(Cmd, globallyAvailableCommands)
}
