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

package download

import (
	"github.com/spf13/cobra"

	"github.com/openshift/moactl/cmd/download/oc"
)

var Cmd = &cobra.Command{
	Use:   "download RESOURCE [flags]",
	Short: "Download necessary tools for using your cluster",
	Long:  "Download necessary tools for using your cluster",
}

func init() {
	Cmd.AddCommand(oc.Cmd)
}
