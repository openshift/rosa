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

package logout

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
)

var Cmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out",
	Long:  "Log out, removing the configuration file.",
	RunE:  run,
}

func run(cmd *cobra.Command, argv []string) error {
	// Remove the configuration file:
	err := ocm.Remove()
	if err != nil {
		return fmt.Errorf("Failed to remove config file: %v", err)
	}

	return nil
}
