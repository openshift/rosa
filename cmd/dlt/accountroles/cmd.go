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

package accountroles

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
	mode   string
	prefix string
}

var Cmd = &cobra.Command{
	Use:     "account-roles",
	Aliases: []string{"accountroles", "accountrole", "account-role"},
	Short:   "Delete Account Roles",
	Long:    "Cleans up account roles from the current AWS account.",
	Example: `  # Delete Account roles"
  rosa delete account-roles -p prefix`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.mode,
		"mode",
		modes[0],
		"How to perform the operation. Valid options are:\n"+
			"auto: Account roles will be deleted automatically using the current AWS account\n"+
			"manual: Command to delete the account roles will be output which can be used to delete manually",
	)
	Cmd.RegisterFlagCompletionFunc("mode", modeCompletion)

	flags.StringVarP(
		&args.prefix,
		"prefix",
		"p",
		"",
		"Prefix of the account roles to be deleted.",
	)
	Cmd.MarkFlagRequired("prefix")
	confirm.AddFlag(flags)
}

func modeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return modes, cobra.ShellCompDirectiveDefault
}

func run(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

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

	env, err := ocm.GetEnv()
	if err != nil {
		reporter.Errorf("Error getting environment %s", err)
		os.Exit(1)
	}
	creator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get IAM credentials: %s", err)
		os.Exit(1)
	}
	clusters, err := ocmClient.GetAllClusters(creator)
	if err != nil {
		reporter.Errorf("Error getting clusters %s", err)
		os.Exit(1)
	}
	finalRoleList := []string{}
	roles, err := awsClient.GetAccountRoleForCurrentEnvWithPrefix(env, args.prefix)
	if err != nil {
		reporter.Errorf("Error getting role: %s", err)
		os.Exit(1)
	}
	if len(roles) == 0 {
		reporter.Errorf("There are no roles to be deleted")
		os.Exit(1)
	}
	for _, role := range roles {
		if role.RoleName == "" {
			continue
		}
		clusterID := checkIfRoleAssociated(clusters, role)
		if clusterID != "" {
			reporter.Errorf("Role %s is associated with the cluster %s", role.RoleName, clusterID)
			os.Exit(1)
		}
		finalRoleList = append(finalRoleList, role.RoleName)
	}

	if len(finalRoleList) == 0 {
		reporter.Errorf("There are no roles to be deleted")
		os.Exit(1)
	}

	// Determine if interactive mode is needed
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") {
		interactive.Enable()
	}
	mode := args.mode
	if interactive.Enabled() {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "Account role deletion mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  mode,
			Options:  modes,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid Account role deletion mode: %s", err)
			os.Exit(1)
		}
	}
	switch mode {
	case "auto":
		ocmClient.LogEvent("ROSADeleteAccountRoleModeAuto")
		for _, role := range finalRoleList {
			if !confirm.Prompt(true, "Delete the account role '%s'?", role) {
				continue
			}
			err := awsClient.DeleteAccountRole(role)
			if err != nil {
				reporter.Errorf("There was an error deleting the account roles: %s", err)
				continue
			}
		}
	case "manual":
		ocmClient.LogEvent("ROSADeleteAccountRoleModeManual")
		policyMap, err := awsClient.GetAccountRolePolicies(finalRoleList)
		if err != nil {
			reporter.Errorf("There was an error getting the policy: %v", err)
			os.Exit(1)
		}
		instanceProfileRoles, err := awsClient.GetInstanceProfilesForRoles(finalRoleList)
		if err != nil {
			reporter.Errorf("There was an error getting the instance policy: %v", err)
			os.Exit(1)
		}

		commands := buildCommand(finalRoleList, policyMap, instanceProfileRoles)

		if reporter.IsTerminal() {
			reporter.Infof("Run the following commands to delete the account roles:\n")
		}
		fmt.Println(commands)
	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", modes)
		os.Exit(1)
	}
}

func checkIfRoleAssociated(clusters []*cmv1.Cluster, role aws.Role) string {
	for _, cluster := range clusters {
		if cluster.AWS().STS().RoleARN() == role.RoleARN ||
			cluster.AWS().STS().SupportRoleARN() == role.RoleARN ||
			cluster.AWS().STS().InstanceIAMRoles().MasterRoleARN() == role.RoleARN ||
			cluster.AWS().STS().InstanceIAMRoles().WorkerRoleARN() == role.RoleARN {
			return cluster.Name()
		}
	}
	return ""
}

func buildCommand(roleNames []string, policyMap map[string][]aws.PolicyDetail,
	instanceProfileRoles map[string][]string) string {
	commands := []string{}
	for _, roleName := range roleNames {
		policyDetails := policyMap[roleName]
		for _, policyDetail := range policyDetails {
			if policyDetail.PolicType == aws.Attached && policyDetail.PolicyArn != "" {
				deletePolicy := fmt.Sprintf("\taws iam detach-role-policy --role-name  %s  --policy-arn  %s",
					roleName, policyDetail.PolicyArn)
				commands = append(commands, deletePolicy)
			}
			if policyDetail.PolicType == aws.Inline && policyDetail.PolicyName != "" {
				deletePolicy := fmt.Sprintf("\taws iam delete-role-policy --role-name  %s  --policy-name  %s",
					roleName, policyDetail.PolicyName)
				commands = append(commands, deletePolicy)
			}
		}
		instanceProfiles := instanceProfileRoles[roleName]
		for _, instanceProfile := range instanceProfiles {
			removePolicy := fmt.Sprintf("\taws iam remove-role-from-instance-profile --role-name  %s  "+
				"--instance-profile-name  %s",
				roleName, instanceProfile)
			commands = append(commands, removePolicy)
		}
		deleteRole := fmt.Sprintf("\taws iam delete-role --role-name  %s", roleName)
		commands = append(commands, deleteRole)
	}
	return strings.Join(commands, "\n")
}
