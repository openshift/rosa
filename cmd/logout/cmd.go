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
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/config"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	hyperfleet bool
}

var Cmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out",
	Long:  "Log out, removing the configuration file. Use --hyperfleet to clear only the Platform API configuration.",
	Run:   run,
	Args:  cobra.NoArgs,
}

func init() {
	Cmd.Flags().BoolVar(&args.hyperfleet, "hyperfleet", false,
		"Clear the Platform API configuration only, keeping OCM credentials intact")
}

func run(_ *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporter()
	err := runLogout(reporter, args.hyperfleet)
	if err != nil {
		reporter.Errorf("Failed to log out: %v", err)
		os.Exit(1)
	}
}

func runLogout(reporter *rprtr.Object, hyperfleet bool) error {
	if hyperfleet {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %v", err)
		}
		if cfg == nil || cfg.HyperfleetURL == "" {
			reporter.Infof("Not logged in to Platform API")
			return nil
		}
		cfg.HyperfleetURL = ""
		if err = config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save configuration: %v", err)
		}
		reporter.Infof("Logged out from Platform API")
		return nil
	}
	return config.Remove()
}
