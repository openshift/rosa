/*
Copyright (c) 2021 Red Hat, Inc.

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

package operatorrole

import (
	"fmt"

	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:     "operator-roles",
	Aliases: []string{"operatorrole"},
	Short:   "Delete Operator Roles",
	Long:    "Cleans up operator roles of deleted STS cluster.",
	Example: `  # Delete Operator roles for cluster named "mycluster"
  rosa delete operator-roles --cluster=mycluster`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"ID or Name of the cluster (deleted/archived) to delete the operator roles from (required).",
	)
	aws.AddModeFlag(Cmd)
	confirm.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.NewLogger()

	if len(argv) == 1 && !cmd.Flag("cluster").Changed {
		args.clusterKey = argv[0]
	}

	mode, err := aws.GetMode()
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := args.clusterKey
	if !ocm.IsValidClusterKey(clusterKey) {
		reporter.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
		os.Exit(1)
	}

	// Determine if interactive mode is needed
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") {
		interactive.Enable()
	}

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create AWS client: %v", err)
		os.Exit(1)
	}
	creator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get IAM credentials: %s", err)
		os.Exit(1)
	}
	// Create the client for the OCM API:
	ocmClient, err := ocm.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	reporter.Debugf("Loading cluster '%s'", clusterKey)
	sub, err := ocmClient.GetClusterUsingSubscription(clusterKey, creator)
	if err != nil {
		if errors.GetType(err) == errors.Conflict {
			reporter.Errorf("More than one cluster found with the same name '%s'. Please "+
				"use cluster ID instead", clusterKey)
			os.Exit(1)
		}
		reporter.Errorf("Error validating cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}
	clusterID := clusterKey
	if sub != nil {
		clusterID = sub.ClusterID()
	}
	c, err := ocmClient.GetClusterByID(clusterID, creator)
	if err != nil {
		if errors.GetType(err) != errors.NotFound {
			reporter.Errorf("Error validating cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
	}

	if c != nil && c.ID() != "" {
		reporter.Errorf("Cluster '%s' is in '%s' state. Operator roles can be deleted only for the "+
			"uninstalled clusters", c.ID(), c.State())
		os.Exit(1)
	}

	env, err := ocm.GetEnv()
	if err != nil {
		reporter.Errorf("Error getting environment %s", err)
		os.Exit(1)
	}
	if env != "production" {
		if !confirm.Prompt(true, "You are running delete operation from staging. Please ensure "+
			"there are no clusters using these operator roles in the production. Are you sure you want to proceed?") {
			os.Exit(1)
		}
	}

	if interactive.Enabled() {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "Operator roles deletion mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  aws.ModeAuto,
			Options:  aws.Modes,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid operator role deletion mode: %s", err)
			os.Exit(1)
		}
	}
	var spin *spinner.Spinner
	if reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		reporter.Infof("Fetching operator roles for the cluster: %s", clusterKey)
		spin.Start()
	}

	credRequests, err := ocmClient.GetCredRequests()
	if err != nil {
		reporter.Errorf("Error getting operator credential request from OCM %s", err)
		os.Exit(1)
	}

	roles, err := awsClient.GetOperatorRolesFromAccount(sub.ClusterID(), credRequests)
	if len(roles) == 0 {
		if spin != nil {
			spin.Stop()
		}
		reporter.Infof("There are no operator roles to delete for the cluster '%s'", clusterKey)
		return
	}
	if spin != nil {
		spin.Stop()
	}

	switch mode {
	case aws.ModeAuto:
		ocmClient.LogEvent("ROSADeleteOperatorroleModeAuto", nil)
		for _, role := range roles {
			if !confirm.Prompt(true, "Delete the operator roles  '%s'?", role) {
				continue
			}
			err = awsClient.DeleteOperatorRole(role)
			if err != nil {
				reporter.Errorf("There was an error deleting the Operator Roles: %s", err)
				continue
			}
		}
		reporter.Infof("Successfully deleted the operator roles")
	case aws.ModeManual:
		ocmClient.LogEvent("ROSADeleteOperatorroleModeManual", nil)
		policyMap, err := awsClient.GetPolicies(roles)
		if err != nil {
			reporter.Errorf("There was an error getting the policy: %v", err)
			os.Exit(1)
		}
		commands := buildCommand(roles, policyMap)
		if reporter.IsTerminal() {
			reporter.Infof("Run the following commands to delete the Operator roles:\n")
		}
		fmt.Println(commands)
	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
}

func buildCommand(roleNames []string, policyMap map[string][]string) string {
	commands := []string{}
	for _, roleName := range roleNames {
		policyARN := policyMap[roleName]
		detachPolicy := ""
		if len(policyARN) > 0 {
			detachPolicy = fmt.Sprintf("\taws iam detach-role-policy --role-name  %s --policy-arn  %s",
				roleName, policyARN[0])
		}
		deleteRole := fmt.Sprintf("\taws iam delete-role --role-name  %s", roleName)
		commands = append(commands, detachPolicy, deleteRole)
	}
	return strings.Join(commands, "\n")
}
