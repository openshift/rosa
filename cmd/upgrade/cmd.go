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

package upgrade

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/upgrade/accountroles"
	"github.com/openshift/rosa/cmd/upgrade/cluster"
	"github.com/openshift/rosa/cmd/upgrade/machinepool"
	"github.com/openshift/rosa/cmd/upgrade/operatorroles"
	"github.com/openshift/rosa/cmd/upgrade/roles"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/interactive"
)

var Cmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade a resource",
	Long:  "Upgrade a resource",
	Args:  cobra.NoArgs,
}

func init() {
	Cmd.AddCommand(cluster.Cmd)
	Cmd.AddCommand(machinepool.Cmd)
	Cmd.AddCommand(accountroles.Cmd)
	Cmd.AddCommand(operatorroles.Cmd)
	Cmd.AddCommand(roles.Cmd)

	flags := Cmd.PersistentFlags()
	arguments.AddProfileFlag(flags)
	arguments.AddRegionFlag(flags)
	interactive.AddFlag(flags)

	globallyAvailableCommands := []*cobra.Command{
		accountroles.Cmd, operatorroles.Cmd,
		roles.Cmd, machinepool.Cmd, cluster.Cmd,
	}
	arguments.MarkRegionDeprecated(Cmd, globallyAvailableCommands)
}
