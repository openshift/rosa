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

package installation

import (
	"fmt"
	"os"

	asv1 "github.com/openshift-online/ocm-sdk-go/addonsmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "addon-installation clusterID AddonInstallationID",
	Aliases: []string{"add-on-installation"},
	Short:   "Show details of an add-on installation",
	Long:    "Show details of an add-on installation",
	Example: `  # Describe the 'bar' add-on installation on cluster 'foo'
  rosa describe addon-installation --cluster foo --addon bar`,
	Run:  run,
	Args: cobra.NoArgs,
}

var args struct {
	clusterKey      string
	installationKey string
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to list the add-ons of (required).",
	)

	flags.StringVar(
		&args.installationKey,
		"addon",
		"",
		"Name or ID of the addon installation (required).",
	)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	if args.clusterKey == "" {
		r.Reporter.Errorf(
			"Expected the cluster to be specified with the --cluster flag")
		os.Exit(1)
	}
	ocm.SetClusterKey(args.clusterKey)

	if args.installationKey == "" {
		r.Reporter.Errorf(
			"Expected the add-on installation to be specified with the --addon flag")
		os.Exit(1)
	}

	if err := describeAddonInstallation(r, args.installationKey); err != nil {
		r.Reporter.Errorf("Failed to describe add-on installation: %v", err)
		os.Exit(1)
	}
}

func describeAddonInstallation(r *rosa.Runtime, installationKey string) error {
	cluster := r.FetchCluster()

	installation, err := r.OCMClient.GetAddOnInstallation(cluster.ID(), installationKey)
	if err != nil {
		return err
	}

	fmt.Printf(`%-28s %s
%-28s %s
%-28s %s
`,
		"Id:", installation.ID(),
		"Href:", installation.HREF(),
		"Addon state:", installation.State(),
	)

	parameters := installation.Parameters()
	if parameters.Len() > 0 {
		fmt.Println("Parameters:")
	}
	parameters.Each(func(parameter *asv1.AddonInstallationParameter) bool {
		fmt.Printf("\t%-28q: %q\n", parameter.Id(), parameter.Value())
		return true
	})

	return nil
}
