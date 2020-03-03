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

package create

import (
	"fmt"
	"os"

	"github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/service/moactl/pkg/debug"
	rprtr "gitlab.cee.redhat.com/service/moactl/pkg/reporter"
)

var args struct {
	region string
}

var Cmd = &cobra.Command{
	Use:   "create [FLAGS] NAME",
	Short: "Create cluster",
	Long:  "Create cluster.",
	Run:   run,
}

func init() {
	fs := Cmd.Flags()
	fs.StringVar(
		&args.region,
		"region",
		"us-east-1",
		"Region to create the cluster in.",
	)
}

func run(cmd *cobra.Command, argv []string) {
	// Create the reporter:
	reporter, err := rprtr.New().
		Build()
	if err != nil {
		fmt.Errorf("Can't create reporter: %v\n", err)
		os.Exit(1)
	}

	// Check command line arguments:
	if len(argv) != 1 {
		reporter.Errorf(
			"Expected exactly one command line parameter containing the name " +
				"of the cluster",
		)
		os.Exit(1)
	}
	name := argv[0]

	// Check the command line options:
	if args.region == "" {
		reporter.Errorf("Option '--region' is mandatory")
		os.Exit(1)
	}

	// Check that there is an OCM token in the environment. This will not be needed once we are
	// able to derive OCM credentials from AWS credentials.
	ocmToken := os.Getenv("OCM_TOKEN")
	if ocmToken == "" {
		reporter.Errorf("Environment variable 'OCM_TOKEN' is not set")
		os.Exit(1)
	}

	// Create the logger that will be used by the OCM connection:
	ocmLogger, err := sdk.NewStdLoggerBuilder().
		Debug(debug.Enabled()).
		Build()
	if err != nil {
		reporter.Errorf("Can't create OCM logger: %v", err)
		os.Exit(1)
	}

	// Create the client for the OCM API:
	ocmConnection, err := sdk.NewConnectionBuilder().
		Logger(ocmLogger).
		Tokens(ocmToken).
		URL("https://api-integration.6943.hive-integration.openshiftapps.com").
		Build()
	if err != nil {
		reporter.Errorf("Can't create OCM connection: %v", err)
		os.Exit(1)
	}

	// Create the permissions needed to create the cluster:
	reporter.Infof("Creating permissions")

	// Create the cluster:
	reporter.Infof("Creating cluster '%s'", name)
	ocmCluster, err := cmv1.NewCluster().
		Name(name).
		Region(
			cmv1.NewCloudRegion().
				ID(args.region),
		).
		Build()
	if err != nil {
		reporter.Errorf("Can't create description of OCM cluster: %v", err)
		os.Exit(1)
	}
	_, err = ocmConnection.ClustersMgmt().V1().Clusters().Add().
		Body(ocmCluster).
		Send()
	if err != nil {
		reporter.Infof("Can't create OCM cluster: %v", err)
		os.Exit(1)
	}
	reporter.Infof("Cluster '%s' is being created", name)
}
