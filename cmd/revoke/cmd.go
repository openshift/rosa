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

package revoke

import (
	"github.com/spf13/cobra"

	"github.com/openshift/moactl/cmd/revoke/user"
	"github.com/openshift/moactl/pkg/confirm"
)

var Cmd = &cobra.Command{
	Use:   "revoke RESOURCE [flags]",
	Short: "Revoke role from a specific resource",
	Long:  "Revoke role from a specific resource",
}

func init() {
	flags := Cmd.PersistentFlags()
	confirm.AddFlag(flags)

	Cmd.AddCommand(user.Cmd)
}
