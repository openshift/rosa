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

package edit

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/edit/addon"
	"github.com/openshift/rosa/cmd/edit/autoscaler"
	"github.com/openshift/rosa/cmd/edit/cluster"
	"github.com/openshift/rosa/cmd/edit/imagemirror"
	"github.com/openshift/rosa/cmd/edit/ingress"
	"github.com/openshift/rosa/cmd/edit/kubeletconfig"
	"github.com/openshift/rosa/cmd/edit/machinepool"
	"github.com/openshift/rosa/cmd/edit/service"
	"github.com/openshift/rosa/cmd/edit/tuningconfigs"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
)

var Cmd = &cobra.Command{
	Use:     "edit",
	Aliases: []string{"update"},
	Short:   "Edit a specific resource",
	Long:    "Edit a specific resource",
	Args:    cobra.NoArgs,
}

func init() {
	Cmd.AddCommand(addon.Cmd)
	Cmd.AddCommand(cluster.Cmd)
	Cmd.AddCommand(ingress.Cmd)
	Cmd.AddCommand(service.Cmd)
	Cmd.AddCommand(tuningconfigs.Cmd)
	autoscalerCommand := autoscaler.NewEditAutoscalerCommand()
	Cmd.AddCommand(autoscalerCommand)
	kubeletConfig := kubeletconfig.NewEditKubeletConfigCommand()
	Cmd.AddCommand(kubeletConfig)
	imageMirrorCommand := imagemirror.NewEditImageMirrorCommand()
	Cmd.AddCommand(imageMirrorCommand)

	flags := Cmd.PersistentFlags()
	arguments.AddProfileFlag(flags)
	arguments.AddRegionFlag(flags)
	interactive.AddFlag(flags)
	confirm.AddFlag(flags)

	machinepoolCommand := machinepool.NewEditMachinePoolCommand()
	Cmd.AddCommand(machinepoolCommand)
	globallyAvailableCommands := []*cobra.Command{
		autoscalerCommand, addon.Cmd,
		service.Cmd, cluster.Cmd,
		imageMirrorCommand, ingress.Cmd, kubeletConfig,
		machinepoolCommand, tuningconfigs.Cmd,
	}
	arguments.MarkRegionDeprecated(Cmd, globallyAvailableCommands)
}
