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

package grant

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/grant/roleBinding"
	"github.com/openshift/rosa/cmd/grant/user"
	"github.com/openshift/rosa/pkg/arguments"
)

var Cmd = &cobra.Command{
	Use:   "grant",
	Short: "Grant role to a specific resource",
	Long:  "Grant role to a specific resource",
}

func init() {
	Cmd.AddCommand(user.Cmd)
	Cmd.AddCommand(roleBinding.Cmd)

	flags := Cmd.PersistentFlags()
	arguments.AddProfileFlag(flags)
	arguments.AddRegionFlag(flags)
}
