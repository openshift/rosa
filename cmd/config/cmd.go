/*
Copyright (c) 2024 Red Hat, Inc.

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

package config

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/config/get"
	"github.com/openshift/rosa/cmd/config/set"
	"github.com/openshift/rosa/pkg/config"
)

func longHelp() string {
	loc, err := config.Location()
	if err != nil {
		// I think this only happens if homedir.Dir() fails, which is unlikely.
		loc = fmt.Sprintf("UNKNOWN (%s)", err)
	}
	return fmt.Sprintf(`Get or set variables from a configuration file.

The location of the configuration file is gleaned from the 'OCM_CONFIG' environment variable,
or ~/.ocm.json if that variable is unset. Currently using: %s

The following variables are supported:

%s

Note that "rosa config get access_token" gives whatever the file contains - may be missing or expired;
you probably want "rosa token" command instead which will obtain a fresh token if needed.
`, loc, strings.Join(config.ConfigVarDocs(), "\n"))
}

func NewConfigCommand() *cobra.Command {
	Cmd := &cobra.Command{
		Use:   "config COMMAND VARIABLE",
		Short: "get or set configuration variables",
		Long:  longHelp(),
		Args:  cobra.NoArgs,
	}
	Cmd.AddCommand(get.Cmd)
	Cmd.AddCommand(set.Cmd)
	return Cmd
}

var Cmd = NewConfigCommand()
