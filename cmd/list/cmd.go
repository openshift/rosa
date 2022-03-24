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

package list

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/list/accountroles"
	"github.com/openshift/rosa/cmd/list/addon"
	"github.com/openshift/rosa/cmd/list/cluster"
	"github.com/openshift/rosa/cmd/list/gates"
	"github.com/openshift/rosa/cmd/list/idp"
	"github.com/openshift/rosa/cmd/list/ingress"
	"github.com/openshift/rosa/cmd/list/instancetypes"
	"github.com/openshift/rosa/cmd/list/machinepool"
	"github.com/openshift/rosa/cmd/list/ocmroles"
	"github.com/openshift/rosa/cmd/list/region"
	"github.com/openshift/rosa/cmd/list/upgrade"
	"github.com/openshift/rosa/cmd/list/user"
	"github.com/openshift/rosa/cmd/list/userroles"
	"github.com/openshift/rosa/cmd/list/version"
	"github.com/openshift/rosa/cmd/list/service"
	"github.com/openshift/rosa/pkg/arguments"
)

var Cmd = &cobra.Command{
	Use:   "list",
	Short: "List all resources of a specific type",
	Long:  "List all resources of a specific type",
}

func init() {
	Cmd.AddCommand(addon.Cmd)
	Cmd.AddCommand(cluster.Cmd)
	Cmd.AddCommand(gates.Cmd)
	Cmd.AddCommand(idp.Cmd)
	Cmd.AddCommand(ingress.Cmd)
	Cmd.AddCommand(machinepool.Cmd)
	Cmd.AddCommand(region.Cmd)
	Cmd.AddCommand(upgrade.Cmd)
	Cmd.AddCommand(user.Cmd)
	Cmd.AddCommand(version.Cmd)
	Cmd.AddCommand(instancetypes.Cmd)
	Cmd.AddCommand(accountroles.Cmd)
	Cmd.AddCommand(ocmroles.Cmd)
	Cmd.AddCommand(userroles.Cmd)
	Cmd.AddCommand(service.Cmd)
	flags := Cmd.PersistentFlags()
	arguments.AddProfileFlag(flags)
	arguments.AddRegionFlag(flags)
}
