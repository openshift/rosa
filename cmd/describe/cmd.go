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

package describe

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/describe/addon"
	"github.com/openshift/rosa/cmd/describe/admin"
	"github.com/openshift/rosa/cmd/describe/cluster"
	"github.com/openshift/rosa/cmd/describe/installation"
	"github.com/openshift/rosa/cmd/describe/service"
	"github.com/openshift/rosa/cmd/describe/upgrade"
	"github.com/openshift/rosa/pkg/arguments"
)

var Cmd = &cobra.Command{
	Use:   "describe",
	Short: "Show details of a specific resource",
	Long:  "Show details of a specific resource",
}

func init() {
	Cmd.AddCommand(addon.Cmd)
	Cmd.AddCommand(admin.Cmd)
	Cmd.AddCommand(cluster.Cmd)
	Cmd.AddCommand(service.Cmd)
	Cmd.AddCommand(installation.Cmd)
	Cmd.AddCommand(upgrade.Cmd)

	flags := Cmd.PersistentFlags()
	arguments.AddProfileFlag(flags)
	arguments.AddRegionFlag(flags)
}
