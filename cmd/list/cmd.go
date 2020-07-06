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

	"github.com/openshift/moactl/cmd/list/addon"
	"github.com/openshift/moactl/cmd/list/cluster"
	"github.com/openshift/moactl/cmd/list/idp"
	"github.com/openshift/moactl/cmd/list/ingress"
	"github.com/openshift/moactl/cmd/list/user"
)

var Cmd = &cobra.Command{
	Use:   "list RESOURCE",
	Short: "List all resources of a specific type",
	Long:  "List all resources of a specific type",
}

func init() {
	Cmd.AddCommand(addon.Cmd)
	Cmd.AddCommand(cluster.Cmd)
	Cmd.AddCommand(idp.Cmd)
	Cmd.AddCommand(ingress.Cmd)
	Cmd.AddCommand(user.Cmd)
}
