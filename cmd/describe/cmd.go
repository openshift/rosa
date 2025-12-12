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

	"github.com/openshift/rosa/cmd/describe/accessrequest"
	"github.com/openshift/rosa/cmd/describe/addon"
	"github.com/openshift/rosa/cmd/describe/admin"
	"github.com/openshift/rosa/cmd/describe/autoscaler"
	"github.com/openshift/rosa/cmd/describe/breakglasscredential"
	"github.com/openshift/rosa/cmd/describe/cluster"
	"github.com/openshift/rosa/cmd/describe/externalauthprovider"
	"github.com/openshift/rosa/cmd/describe/iamserviceaccount"
	"github.com/openshift/rosa/cmd/describe/ingress"
	"github.com/openshift/rosa/cmd/describe/installation"
	"github.com/openshift/rosa/cmd/describe/kubeletconfig"
	"github.com/openshift/rosa/cmd/describe/logforwarders"
	"github.com/openshift/rosa/cmd/describe/machinepool"
	"github.com/openshift/rosa/cmd/describe/service"
	"github.com/openshift/rosa/cmd/describe/tuningconfigs"
	"github.com/openshift/rosa/cmd/describe/upgrade"
	"github.com/openshift/rosa/pkg/arguments"
)

var Cmd = &cobra.Command{
	Use:   "describe",
	Short: "Show details of a specific resource",
	Long:  "Show details of a specific resource",
	Args:  cobra.NoArgs,
}

func init() {
	machinePoolCommand := machinepool.NewDescribeMachinePoolCommand()
	ingressCommand := ingress.NewDescribeIngressCommand()
	kubeletconfig := kubeletconfig.NewDescribeKubeletConfigCommand()
	accessrequestCommand := accessrequest.NewDescribeAccessRequestCommand()
	cmds := []*cobra.Command{
		addon.Cmd, admin.Cmd, cluster.Cmd, iamserviceaccount.Cmd, service.Cmd,
		installation.Cmd, upgrade.Cmd, tuningconfigs.Cmd,
		machinePoolCommand, kubeletconfig,
		autoscaler.NewDescribeAutoscalerCommand(), ingressCommand,
		externalauthprovider.Cmd, breakglasscredential.Cmd,
		accessrequestCommand, logforwarders.NewDescribeLogForwarderCommand(),
	}
	for _, cmd := range cmds {
		Cmd.AddCommand(cmd)
	}

	flags := Cmd.PersistentFlags()
	arguments.AddProfileFlag(flags)
	arguments.AddRegionFlag(flags)

	globallyAvailableCommands := []*cobra.Command{
		tuningconfigs.Cmd, cluster.Cmd, service.Cmd,
		machinePoolCommand, addon.Cmd, upgrade.Cmd,
		admin.Cmd, breakglasscredential.Cmd,
		externalauthprovider.Cmd, installation.Cmd,
		kubeletconfig, upgrade.Cmd, ingressCommand,
		accessrequestCommand,
	}
	arguments.MarkRegionDeprecated(Cmd, globallyAvailableCommands)
}
