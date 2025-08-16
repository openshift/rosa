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
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/dlt/accountroles"
	"github.com/openshift/rosa/cmd/dlt/admin"
	"github.com/openshift/rosa/cmd/dlt/autoscaler"
	"github.com/openshift/rosa/cmd/dlt/cluster"
	"github.com/openshift/rosa/cmd/dlt/dnsdomains"
	"github.com/openshift/rosa/cmd/dlt/externalauthprovider"
	"github.com/openshift/rosa/cmd/dlt/iamserviceaccount"
	"github.com/openshift/rosa/cmd/dlt/idp"
	"github.com/openshift/rosa/cmd/dlt/ingress"
	"github.com/openshift/rosa/cmd/dlt/kubeletconfig"
	"github.com/openshift/rosa/cmd/dlt/machinepool"
	"github.com/openshift/rosa/cmd/dlt/ocmrole"
	"github.com/openshift/rosa/cmd/dlt/oidcconfig"
	"github.com/openshift/rosa/cmd/dlt/oidcprovider"
	"github.com/openshift/rosa/cmd/dlt/operatorrole"
	"github.com/openshift/rosa/cmd/dlt/service"
	"github.com/openshift/rosa/cmd/dlt/tuningconfigs"
	"github.com/openshift/rosa/cmd/dlt/upgrade"
	"github.com/openshift/rosa/cmd/dlt/userrole"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/interactive/confirm"
)

var Cmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"remove"},
	Short:   "Delete a specific resource",
	Long:    "Delete a specific resource",
	Args:    cobra.NoArgs,
}

func init() {
	Cmd.AddCommand(admin.Cmd)
	Cmd.AddCommand(cluster.Cmd)
	Cmd.AddCommand(iamserviceaccount.Cmd)
	Cmd.AddCommand(idp.Cmd)
	Cmd.AddCommand(ingress.Cmd)
	machinepoolCommand := machinepool.NewDeleteMachinePoolCommand()
	Cmd.AddCommand(machinepoolCommand)
	Cmd.AddCommand(upgrade.Cmd)
	Cmd.AddCommand(oidcconfig.Cmd)
	Cmd.AddCommand(oidcprovider.Cmd)
	Cmd.AddCommand(operatorrole.Cmd)
	Cmd.AddCommand(accountroles.Cmd)
	Cmd.AddCommand(ocmrole.Cmd)
	Cmd.AddCommand(userrole.Cmd)
	Cmd.AddCommand(service.Cmd)
	Cmd.AddCommand(tuningconfigs.Cmd)
	Cmd.AddCommand(dnsdomains.Cmd)
	autoscalerCommand := autoscaler.NewDeleteAutoscalerCommand()
	Cmd.AddCommand(autoscalerCommand)
	kubeletconfig := kubeletconfig.NewDeleteKubeletConfigCommand()
	Cmd.AddCommand(kubeletconfig)
	Cmd.AddCommand(externalauthprovider.Cmd)

	flags := Cmd.PersistentFlags()
	arguments.AddProfileFlag(flags)
	arguments.AddRegionFlag(flags)
	confirm.AddFlag(flags)

	globallyAvailableCommands := []*cobra.Command{
		accountroles.Cmd, operatorrole.Cmd,
		userrole.Cmd, ocmrole.Cmd,
		oidcprovider.Cmd, upgrade.Cmd, admin.Cmd,
		service.Cmd, autoscalerCommand, iamserviceaccount.Cmd, idp.Cmd,
		cluster.Cmd, dnsdomains.Cmd, externalauthprovider.Cmd,
		kubeletconfig, machinepoolCommand, tuningconfigs.Cmd,
	}
	arguments.MarkRegionDeprecated(Cmd, globallyAvailableCommands)
}
