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

package get

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	Writer io.Writer = os.Stdout
)

var Cmd = NewConfigGetCommand()

func NewConfigGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get [flags] VARIABLE",
		Short: "Prints the value of a config variable",
		Long:  "Prints the value of a config variable. See 'rosa config --help' for supported config variables.",
		Args:  cobra.ExactArgs(1),
		Run:   run,
	}
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime()

	err := PrintConfig(argv[0])
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}
}

func PrintConfig(arg string) error {
	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("Failed to load config file: %v", err)
	}

	// If the configuration file doesn't exist yet assume that all the configuration settings
	// are empty:
	if cfg == nil {
		loc, err := config.Location()
		if err != nil {
			return fmt.Errorf("Failed to find config file location: %v", err)
		}
		return fmt.Errorf("Config file '%s' does not exist", loc)
	}

	// Print the value of the requested configuration setting:
	switch arg {
	case "access_token":
		fmt.Fprintf(Writer, "%s\n", cfg.AccessToken)
	case "client_id":
		fmt.Fprintf(Writer, "%s\n", cfg.ClientID)
	case "client_secret":
		fmt.Fprintf(Writer, "%s\n", cfg.ClientSecret)
	case "insecure":
		fmt.Fprintf(Writer, "%v\n", cfg.Insecure)
	case "refresh_token":
		fmt.Fprintf(Writer, "%s\n", cfg.RefreshToken)
	case "scopes":
		fmt.Fprintf(Writer, "%s\n", cfg.Scopes)
	case "token_url":
		fmt.Fprintf(Writer, "%s\n", cfg.TokenURL)
	case "url":
		fmt.Fprintf(Writer, "%s\n", cfg.URL)
	case "fedramp":
		fmt.Fprintf(Writer, "%v\n", cfg.FedRAMP)
	default:
		return fmt.Errorf("'%s' is not a supported setting", arg)
	}
	return nil
}
