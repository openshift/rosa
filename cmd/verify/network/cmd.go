/*
Copyright (c) 2023 Red Hat, Inc.

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

package network

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	region     string
	roleArn    string
	statusOnly bool
	subnetIDs  []string
	watch      bool
}

var Cmd = makeCmd()

func makeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "network",
		Short: "Verify VPC subnets are configured correctly",
		Long:  "Verify that the VPC subnets are configured correctly.",
		Example: `  # Verify two subnets
	rosa verify network --subnet-ids subnet-03046a9b92b5014fb,subnet-03046a9c92b5014fb`,
		Run: run,
	}
}

type NetworkVerifyState string

const (
	clusterFlag    = "cluster"
	regionFlag     = "region"
	roleArnFlag    = "role-arn"
	statusOnlyFlag = "status-only"
	subnetIDsFlag  = "subnet-ids"
	watchFlag      = "watch"

	NetworkVerifyPending NetworkVerifyState = "pending"
	NetworkVerifyRunning NetworkVerifyState = "running"
	NetworkVerifyPassed  NetworkVerifyState = "passed"
	NetworkVerifyFailed  NetworkVerifyState = "failed"

	delay = 5 * time.Second
)

func init() {
	initFlags(Cmd)
}

func initFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	ocm.AddOptionalClusterFlag(cmd)

	flags.StringSliceVar(
		&args.subnetIDs,
		subnetIDsFlag,
		nil,
		"The Subnet IDs to verify. "+
			"Format should be a comma-separated list.",
	)

	flags.StringVar(
		&args.region,
		regionFlag,
		"",
		"AWS region where the subnets are assigned to.",
	)

	flags.StringVar(
		&args.roleArn,
		roleArnFlag,
		"",
		"STS Role ARN with get secrets permission.",
	)

	flags.BoolVarP(
		&args.watch,
		watchFlag,
		"w",
		false,
		"Watch network verification progress.",
	)

	flags.BoolVarP(
		&args.statusOnly,
		statusOnlyFlag,
		"s",
		false,
		"Check status of previously submitted subnets.",
	)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()
	err := runWithRuntime(r, cmd)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}
}

func runWithRuntime(r *rosa.Runtime, cmd *cobra.Command) error {
	var cluster *cmv1.Cluster

	// validations
	if cmd.Flags().Changed(clusterFlag) {
		// cluster flow
		cluster = r.FetchCluster()
		if cmd.Flags().Changed(subnetIDsFlag) || cmd.Flags().Changed(roleArnFlag) || cmd.Flags().Changed("region") {
			r.Reporter.Errorf("Please remove '--subnet-ids', '--region' and '--role-arn' flags "+
				"when running 'rosa network verify -c %s'",
				cluster.ID())
			os.Exit(1)
		}
		r.Reporter.Debugf("Received the following cluster: %s", cluster.ID())

		args.subnetIDs = cluster.AWS().SubnetIDs()
		args.region = cluster.Region().ID()
	} else {
		// sts role flow
		if !cmd.Flags().Changed(subnetIDsFlag) || !cmd.Flags().Changed(roleArnFlag) || !cmd.Flags().Changed("region") {
			r.Reporter.Errorf("--subnet-ids, --region, --role-arn flags " +
				"are required when running the network verifier without a cluster")
			os.Exit(1)
		}

		if len(args.subnetIDs) == 0 {
			r.Reporter.Errorf("At least one subnet IDs is required")
			os.Exit(1)
		}

		err := aws.ARNValidator(args.roleArn)
		if err != nil {
			r.Reporter.Errorf("Expected a valid ARN: %s", err)
			os.Exit(1)
		}

		r.Reporter.Debugf("Received the following subnet IDs '%v', role arn, '%s' and region '%s'",
			args.subnetIDs, args.roleArn, args.region)
	}

	// run network verifier
	if r.Reporter.IsTerminal() {
		if cmd.Flags().Changed(statusOnlyFlag) {
			r.Reporter.Infof("Checking the status of the following subnet IDs: %v", args.subnetIDs)
		} else {
			r.Reporter.Infof("Verifying the following subnet IDs are configured correctly: %v", args.subnetIDs)
		}
	}

	if !cmd.Flags().Changed(statusOnlyFlag) {
		if cluster != nil {
			_, err := r.OCMClient.VerifyNetworkCluster(cluster.ID())
			if err != nil {
				r.Reporter.Errorf("Error verifying cluster '%s': %s", cluster.ID(), err)
				os.Exit(1)
			}
		} else {
			_, err := r.OCMClient.VerifyNetworkSubnets(args.roleArn, args.region, args.subnetIDs)
			if err != nil {
				r.Reporter.Errorf("Error verifying subnets '%v': %s", args.subnetIDs, err)
				os.Exit(1)
			}
		}
	}

	if args.watch && len(args.subnetIDs) > 0 {
		var spin *spinner.Spinner
		if r.Reporter.IsTerminal() {
			spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		}
		if spin != nil {
			spin.Start()
		}

		for len(args.subnetIDs) > 0 {
			for i := 0; i < len(args.subnetIDs); i++ {
				subnet := args.subnetIDs[i]
				status, err := r.OCMClient.GetVerifyNetworkSubnet(subnet)
				if err == nil && (status.State() == string(NetworkVerifyPending) ||
					status.State() == string(NetworkVerifyRunning)) {
					continue
				}
				printStatus(r.Reporter, spin, subnet, status, err)

				// Remove completed subnets, no need to check these again
				args.subnetIDs[i] = args.subnetIDs[len(args.subnetIDs)-1]
				args.subnetIDs = args.subnetIDs[:len(args.subnetIDs)-1]
			}

			if len(args.subnetIDs) > 0 {
				time.Sleep(delay)
			}
		}

		if spin != nil {
			spin.Stop()
		}
	} else {
		var pending bool = false
		for i := 0; i < len(args.subnetIDs); i++ {
			subnet := args.subnetIDs[i]
			status, err := r.OCMClient.GetVerifyNetworkSubnet(subnet)
			printStatus(r.Reporter, nil, subnet, status, err)
			if status.State() == string(NetworkVerifyPending) || status.State() == string(NetworkVerifyRunning) {
				pending = true
			}
		}

		if pending {
			output := fmt.Sprintf("Run the following command to wait for verification to all subnets to complete:\n"+
				"rosa verify network --watch --status-only --region %s --subnet-ids %s",
				args.region, strings.Join(args.subnetIDs, ","))
			r.Reporter.Infof(output)
		}
	}

	return nil
}

func printStatus(reporter *reporter.Object, spin *spinner.Spinner, subnet string,
	status *cmv1.SubnetNetworkVerification, err error) {
	if spin != nil {
		spin.Stop()
	}

	if err != nil {
		reporter.Infof("%s: %s", subnet, err.Error())
	} else if status.State() == string(NetworkVerifyFailed) {
		reporter.Infof("%s: %s Unable to verify egress to: %v", subnet, status.State(), status.Details())
	} else {
		reporter.Infof("%s: %s", subnet, status.State())
	}

	if spin != nil {
		spin.Restart()
	}
}
