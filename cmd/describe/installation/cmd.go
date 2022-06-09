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

	"github.com/spf13/cobra"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:     "addon-installation clusterID AddonInstallationID",
	Aliases: []string{"add-on-installation"},
	Short:   "Show details of an add-on installation",
	Long:    "Show details of an add-on installation",
	Example: `  # Describe the 'bar' add-on installation on cluster 'foo'
  rosa describe addon-installation --cluster foo --addon bar`,
	Run: run,
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

func run(_ *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	if args.clusterKey == "" {
		reporter.Errorf(
			"Expected the cluster to be specified with the --cluster flag")
		os.Exit(1)
	}
	if args.installationKey == "" {
		reporter.Errorf(
			"Expected the add-on installation to be specified with the --addon flag")
		os.Exit(1)
	}

	// Create the client for the OCM API:
	ocmClient, err := ocm.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM client: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM client: %v", err)
		}
	}()

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create AWS client: %v", err)
		os.Exit(1)
	}

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get AWS creator: %v", err)
		os.Exit(1)
	}

	if err := describeAddonInstallation(ocmClient, awsCreator, args.clusterKey, args.installationKey); err != nil {
		reporter.Errorf("Failed to describe add-on installation: %v", err)
		os.Exit(1)
	}
}

func describeAddonInstallation(ocmClient *ocm.Client, awsCreator *aws.Creator,
	clusterKey string, installationKey string) error {
	cluster, err := ocmClient.GetCluster(clusterKey, awsCreator)
	if err != nil {
		return err
	}

	installation, err := ocmClient.GetAddOnInstallation(cluster.ID(), installationKey)
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
	parameters.Each(func(parameter *cmv1.AddOnInstallationParameter) bool {
		fmt.Printf("\t%-28q: %q\n", parameter.ID(), parameter.Value())
		return true
	})

	return nil
}
