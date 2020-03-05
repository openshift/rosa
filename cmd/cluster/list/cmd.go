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

package list

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/service/moactl/pkg/debug"
	"gitlab.cee.redhat.com/service/moactl/pkg/properties"
	rprtr "gitlab.cee.redhat.com/service/moactl/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:   "list",
	Short: "List clusters",
	Long:  "List clusters.",
	Run:   run,
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
	if len(argv) != 0 {
		reporter.Errorf("Expected exactly zero command line parameters")
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

	// Create the AWS session:
	awsSession, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		reporter.Errorf("Can't create AWS session: %v", err)
		os.Exit(1)
	}

	// Check that the AWS credentials are available:
	_, err = awsSession.Config.Credentials.Get()
	if err != nil {
		reporter.Errorf("Can't find AWS credentials: %v", err)
		os.Exit(1)
	}

	// Get the clients for the AWS services that we will be using:
	awsSts := sts.New(awsSession)

	// Get the details of the current user:
	getCallerIdentityOutput, err := awsSts.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		reporter.Errorf("Can't get caller identity: %v", err)
		os.Exit(1)
	}
	awsCreatorARN := aws.StringValue(getCallerIdentityOutput.Arn)
	reporter.Infof("ARN of current user is '%s'", awsCreatorARN)

	// Extract the account identifier from the ARN of the user:
	awsCreatorParsedARN, err := arn.Parse(awsCreatorARN)
	if err != nil {
		reporter.Infof("Can't parse user ARN '%s': %v", awsCreatorARN, err)
		os.Exit(1)
	}
	awsCreatorAccountID := awsCreatorParsedARN.AccountID
	reporter.Infof("Account identifier is '%s'", awsCreatorAccountID)

	// Create the client for the OCM API:
	ocmConnection, err := sdk.NewConnectionBuilder().
		Logger(ocmLogger).
		Tokens(ocmToken).
		URL("https://api.stage.openshift.com").
		Build()
	if err != nil {
		reporter.Errorf("Can't create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmConnection.Close()
		if err != nil {
			reporter.Errorf("Can't close OCM connection: %v", err)
		}
	}()

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "ID\tNAME\n")

	// Retrieve the list of clusters:
	ocmQuery := fmt.Sprintf("properties.%s = '%s'", properties.CreatorARN, awsCreatorARN)
	ocmRequest := ocmConnection.ClustersMgmt().V1().Clusters().List().
		Search(ocmQuery)
	page := 1
	size := 100
	for {
		ocmResponse, err := ocmRequest.Page(page).Size(size).Send()
		if err != nil {
			reporter.Errorf("Can't retrieve clusters: %v", err)
			os.Exit(1)
		}
		ocmResponse.Items().Each(func(ocmCluster *cmv1.Cluster) bool {
			fmt.Fprintf(
				writer,
				"%s\t%s\n",
				ocmCluster.ID(),
				ocmCluster.Name(),
			)
			return true
		})
		writer.Flush()
		if ocmResponse.Size() != size {
			break
		}
		page++
	}
}
