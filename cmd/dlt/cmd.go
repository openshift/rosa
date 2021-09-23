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

package dlt

import (
	"github.com/openshift/rosa/cmd/dlt/accountroles"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/dlt/admin"
	"github.com/openshift/rosa/cmd/dlt/cluster"
	"github.com/openshift/rosa/cmd/dlt/idp"
	"github.com/openshift/rosa/cmd/dlt/ingress"
	"github.com/openshift/rosa/cmd/dlt/machinepool"
	"github.com/openshift/rosa/cmd/dlt/upgrade"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/interactive/confirm"
)

var Cmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"remove"},
	Short:   "Delete a specific resource",
	Long:    "Delete a specific resource",
}

func init() {
	Cmd.AddCommand(admin.Cmd)
	Cmd.AddCommand(cluster.Cmd)
	Cmd.AddCommand(idp.Cmd)
	Cmd.AddCommand(ingress.Cmd)
	Cmd.AddCommand(machinepool.Cmd)
	Cmd.AddCommand(upgrade.Cmd)
	Cmd.AddCommand(accountroles.Cmd)
	Cmd.AddCommand()

	flags := Cmd.PersistentFlags()
	arguments.AddProfileFlag(flags)
	arguments.AddRegionFlag(flags)
	confirm.AddFlag(flags)
}
