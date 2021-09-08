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

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/spf13/cobra"
)

var modes []string = []string{"auto", "manual"}

var args struct {
	clusterKey string
	mode       string
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
		"Name or ID of the cluster (deleted/archived) to delete the operator ID from (required).",
	)
	Cmd.MarkFlagRequired("cluster")

	flags.StringVar(
		&args.mode,
		"mode",
		modes[0],
		"How to perform the operation. Valid options are:\n"+
			"auto: Operator roles will be deleted automatically using the current AWS account\n"+
			"manual: Command to delete the operator roles will be output which can be used to delete manually",
	)
	Cmd.RegisterFlagCompletionFunc("mode", modeCompletion)
	confirm.AddFlag(flags)
}

func modeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return modes, cobra.ShellCompDirectiveDefault
}

func run(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

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

	creator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get IAM credentials: %s", err)
		os.Exit(1)
	}
	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := ocmClient.GetArchivedCluster(clusterKey)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			cluster, err = ocmClient.GetCluster(clusterKey, creator)
			if err != nil {
				reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
				os.Exit(1)
			}
			if cluster.ID() != "" {
				reporter.Errorf("Cluster '%s' is in '%s' state. Operator roles can be deleted only for the "+
					"uninstalled clusters", cluster.ID(), cluster.State())
				os.Exit(1)
			}
		}
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	mode := args.mode
	if interactive.Enabled() {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "Operator roles deletion mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  mode,
			Options:  modes,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid operator role deletion mode: %s", err)
			os.Exit(1)
		}
	}
	roleNames := getRoleNames(cluster.AWS().STS().OperatorIAMRoles())
	roles, err := awsClient.GetOperatorRolesFromAccount(roleNames)
	if err != nil {
		reporter.Errorf("Error when fetching the roles from aws account: %v", err)
		os.Exit(1)
	}
	if len(roles) == 0 {
		reporter.Errorf("There are no operator roles to be delete from aws account")
		os.Exit(1)
	}
	switch mode {
	case "auto":
		ocmClient.LogEvent("ROSADeleteOperatorroleModeAuto")
		if !confirm.Prompt(true, "Delete the operator roles for cluster '%s'?", clusterKey) {
			os.Exit(0)
		}
		reporter.Infof("Starting to delete the operator roles of cluster '%s'", clusterKey)
		err = awsClient.DeleteOperatorRoles(roles)
		if err != nil {
			reporter.Errorf("There was an error deleting the Operator Roles: %s", err)
			os.Exit(1)
		}
	case "manual":
		ocmClient.LogEvent("ROSADeleteOperatorroleModeManual")
		policyMap, err := awsClient.GetPolicyForOperatorRole(roles)
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
		reporter.Errorf("Invalid mode. Allowed values are %s", modes)
		os.Exit(1)
	}
}

func buildCommand(roleNames []string, policyMap map[string][]string) string {
	commands := []string{}
	for _, roleName := range roleNames {
		policyARN := policyMap[roleName]
		detachPolicy := ""
		if len(policyARN) > 0 {
			detachPolicy = fmt.Sprintf("aws iam detach-role-policy \\\n"+
				"\t--role-name  %s  --policy-arn  %s  \n\n",
				roleName, policyARN[0])
		}
		deleteRole := fmt.Sprintf("aws iam delete-role \\\n"+
			"\t--role-name  %s \n\n",
			roleName)
		commands = append(commands, detachPolicy, deleteRole)
	}
	return strings.Join(commands, "\n\n")
}

func getRoleNames(operatorIAMRoles []*cmv1.OperatorIAMRole) []string {
	roleNames := []string{}
	for _, role := range operatorIAMRoles {
		s := strings.Split(role.RoleARN(), "/")[1]
		roleNames = append(roleNames, s)
	}
	return roleNames
}
