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
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	region     string
	roleArn    string
	statusOnly bool
	subnetIDs  []string
	watch      bool
	tags       []string
	hostedCp   bool
}

var Cmd = makeCmd()

func makeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "network",
		Short: "Verify VPC subnets are configured correctly",
		Long:  "Verify that the VPC subnets are configured correctly.",
		Example: `  # Verify two subnets
	rosa verify network --subnet-ids subnet-03046a9b92b5014fb,subnet-03046a9c92b5014fb`,
		Run:  run,
		Args: cobra.NoArgs,
	}
}

type NetworkVerifyState string

const (
	clusterFlag    = "cluster"
	roleArnFlag    = "role-arn"
	statusOnlyFlag = "status-only"
	subnetIDsFlag  = "subnet-ids"
	watchFlag      = "watch"
	hostedCpFlag   = "hosted-cp"

	NetworkVerifyPending NetworkVerifyState = "pending"
	NetworkVerifyRunning NetworkVerifyState = "running"
	NetworkVerifyPassed  NetworkVerifyState = "passed"
	NetworkVerifyFailed  NetworkVerifyState = "failed"

	delay = 5 * time.Second
)

func init() {
	initFlags(Cmd)
	output.AddFlag(Cmd)
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

	arguments.AddRegionFlag(flags)

	flags.StringVar(
		&args.roleArn,
		roleArnFlag,
		"",
		"STS Role ARN with get secrets permission.",
	)

	flags.StringSliceVar(
		&args.tags,
		"tags",
		nil,
		"Supply custom tags to the network verifier. Tags will default to cluster tags if a cluster is supplied. "+
			"Tags are comma separated, for example: 'key value, foo bar'",
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

	flags.BoolVar(
		&args.hostedCp,
		hostedCpFlag,
		false,
		"Run network verifier with hosted control plane platform configuration",
	)

	arguments.AddProfileFlag(flags)
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
	var err error
	var platform cmv1.Platform

	if cmd.Flags().Changed(clusterFlag) {
		cluster = r.FetchCluster()
	}

	if !cmd.Flags().Changed(subnetIDsFlag) {
		if cluster != nil {
			if !helper.IsBYOVPC(cluster) {
				return fmt.Errorf(
					"running the network verifier is only supported for BYO VPC clusters")
			}

			args.subnetIDs = cluster.AWS().SubnetIDs()
		} else {
			return fmt.Errorf("at least one subnet IDs is required")
		}
	}

	if args.region, err = getRegion(cmd, cluster); err != nil {
		return err
	}

	if cmd.Flags().Changed(roleArnFlag) {
		err := aws.ARNValidator(args.roleArn)
		if err != nil {
			return fmt.Errorf("expected a valid ARN: %s", err)
		}
	} else if !cmd.Flags().Changed(statusOnlyFlag) {
		if cmd.Flags().Changed(clusterFlag) {
			if cluster == nil {
				cluster = r.FetchCluster()
			}
		} else {
			return fmt.Errorf("%s is required", roleArnFlag)
		}
	}

	r.Reporter.Debugf("Received the following subnetIDs: %v", args.subnetIDs)
	if r.Reporter.IsTerminal() {
		if cmd.Flags().Changed(statusOnlyFlag) {
			r.Reporter.Infof("Checking the status of the following subnet IDs: %v", args.subnetIDs)
		} else {
			r.Reporter.Infof("Verifying the following subnet IDs are configured correctly: %v", args.subnetIDs)
		}
	}

	// Custom tags for network verifier
	_tags := args.tags
	tagsList := map[string]string{}

	if len(_tags) > 0 {
		if err := aws.UserTagValidator(_tags); err != nil {
			return fmt.Errorf("%s", err)
		}
		delim := aws.GetTagsDelimiter(_tags)
		for _, tag := range _tags {
			t := strings.Split(tag, delim)
			tagsList[t[0]] = strings.TrimSpace(t[1])
		}
	}

	if !cmd.Flags().Changed(statusOnlyFlag) {
		if cmd.Flags().Changed(clusterFlag) {
			if cmd.Flags().Changed(hostedCpFlag) {
				return fmt.Errorf("'--hosted-cp' flag is not required when running the network verifier with cluster")
			}
			if cluster == nil {
				cluster = r.FetchCluster()
			}
			_, err := r.OCMClient.VerifyNetworkSubnetsByCluster(cluster.ID(), tagsList)
			if err != nil {
				return fmt.Errorf("error verifying subnets by cluster: %s", err)
			}
		} else {
			// Default platform type set to 'aws-classic'
			platform = cmv1.PlatformAwsClassic
			if args.hostedCp {
				platform = cmv1.PlatformAwsHostedCp
			}
			_, err := r.OCMClient.VerifyNetworkSubnets(args.roleArn, args.region, args.subnetIDs, tagsList, platform)
			if err != nil {
				return fmt.Errorf("error verifying subnets: %s", err)
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
				printStatus(r, spin, subnet, status, err)

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
		pending := false
		for i := 0; i < len(args.subnetIDs); i++ {
			subnet := args.subnetIDs[i]
			status, err := r.OCMClient.GetVerifyNetworkSubnet(subnet)
			printStatus(r, nil, subnet, status, err)
			if status.State() == string(NetworkVerifyPending) || status.State() == string(NetworkVerifyRunning) {
				pending = true
			}
		}

		if pending {
			watchCommandOutput := fmt.Sprintf("Run the following command to wait for verification to all subnets to complete:\n"+
				"rosa verify network --watch --status-only --region %s --subnet-ids %s",
				args.region, strings.Join(args.subnetIDs, ","))
			if output.HasFlag() {
				watchCommandOutput += fmt.Sprintf(" --output %s", output.Output())
			}
			r.Reporter.Infof(watchCommandOutput)
		}
	}

	return nil
}

func printStatus(r *rosa.Runtime, spin *spinner.Spinner, subnet string,
	status *cmv1.SubnetNetworkVerification, err error) {
	if spin != nil {
		spin.Stop()
	}

	if err != nil {
		r.Reporter.Infof("%s: %s", subnet, err.Error())
	} else if status.State() == string(NetworkVerifyFailed) {
		r.Reporter.Infof("%s: %s Unable to verify egress to: %v", subnet, status.State(), status.Details())
	} else if output.HasFlag() {
		err := output.Print(status)
		if err != nil {
			r.Reporter.Debugf("%s: unable to output in %s format - %s", subnet, output.Output(), err.Error())
		}
	} else {
		var tags string
		if len(status.Tags()) > 0 {
			tagsList, err := json.Marshal(status.Tags())
			if err != nil {
				r.Reporter.Debugf("%s: unable to marshal tags - %s", subnet, err.Error())
			}
			tags = string(tagsList)
		}
		r.Reporter.Infof("%s, platform: %s, tags: %v: %s", subnet, status.Platform(), tags, status.State())
	}

	if spin != nil {
		spin.Restart()
	}
}

func getRegion(cmd *cobra.Command, cluster *cmv1.Cluster) (region string, err error) {
	if cmd.Flags().Changed("region") {
		region, err = aws.GetRegion(arguments.GetRegion())
		if err != nil {
			return "", fmt.Errorf("error getting region: %v", err)
		}
		return region, nil
	}
	if cluster != nil {
		return cluster.Region().ID(), nil
	}

	return "", fmt.Errorf("region is required")
}
