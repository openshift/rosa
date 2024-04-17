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

package verify

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/verify/network"
	"github.com/openshift/rosa/cmd/verify/oc"
	"github.com/openshift/rosa/cmd/verify/permissions"
	"github.com/openshift/rosa/cmd/verify/quota"
	"github.com/openshift/rosa/cmd/verify/rosa"
)

var Cmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify resources are configured correctly for cluster install",
	Long:  "Verify resources are configured correctly for cluster install",
	Args:  cobra.NoArgs,
}

func init() {
	Cmd.AddCommand(network.Cmd)
	Cmd.AddCommand(oc.Cmd)
	Cmd.AddCommand(permissions.Cmd)
	Cmd.AddCommand(quota.Cmd)
	Cmd.AddCommand(rosa.NewVerifyRosaCommand())
}
