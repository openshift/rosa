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
	roleName string
	mode     string
	prefix   string
}

var Cmd = &cobra.Command{
	Use:     "account-roles",
	Aliases: []string{"accountroles", "accountrole", "account-role"},
	Short:   "Delete Account Roles",
	Long:    "Cleans up account roles from the current AWS account.",
	Example: `  # Delete Account roles"
  rosa delete account-roles [-r role-name | -p prefix]`,
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
		&args.roleName,
		"role-name",
		"r",
		"",
		"Account role name to be deleted.",
	)

	flags.StringVarP(
		&args.prefix,
		"prefix",
		"p",
		"",
		"Prefix of the account roles to be deleted.",
	)

	confirm.AddFlag(flags)
}

func modeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return modes, cobra.ShellCompDirectiveDefault
}

func run(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	if cmd.Flags().Changed("prefix") && cmd.Flags().Changed("role-name") {
		reporter.Errorf("Provide either account role prefix '-p' or role name '-r' not both")
		os.Exit(1)
	}

	if args.roleName == "" && args.prefix == "" {
		reporter.Errorf("Option account role prefix '-p' or role name '-r' is mandatory")
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
	if args.roleName != "" {
		role, err := awsClient.GetAccountRoleForCurrentEnv(env, args.roleName)
		if err != nil {
			reporter.Errorf("Error getting role: %s", err)
			os.Exit(1)
		}
		if !checkIfRoleExists(clusters, role) {
			finalRoleList = append(finalRoleList, role.RoleName)
		}
	} else if args.prefix != "" {
		roles, err := awsClient.GetAccountRoleForCurrentEnvWithPrefix(env, args.prefix)
		if err != nil {
			reporter.Errorf("Error getting role: %s", err)
			os.Exit(1)
		}
		for _, role := range roles {
			if !checkIfRoleExists(clusters, role) {
				finalRoleList = append(finalRoleList, role.RoleName)
			}
		}
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
		commands := buildCommand(finalRoleList, policyMap)
		if reporter.IsTerminal() {
			reporter.Infof("Run the following commands to delete the account roles:\n")
		}
		fmt.Println(commands)
	}
}

func checkIfRoleExists(clusters []*cmv1.Cluster, role aws.Role) bool {
	for _, cluster := range clusters {
		if cluster.AWS().STS().RoleARN() == role.RoleARN ||
			cluster.AWS().STS().SupportRoleARN() == role.RoleARN ||
			cluster.AWS().STS().InstanceIAMRoles().MasterRoleARN() == role.RoleARN ||
			cluster.AWS().STS().InstanceIAMRoles().WorkerRoleARN() == role.RoleARN {
			return true
		}
	}
	return false
}

func buildCommand(roleNames []string, policyMap map[string]string) string {
	commands := []string{}
	for _, roleName := range roleNames {
		deletePolicy := ""
		if policyMap[roleName] != "" {
			policies := strings.Split(policyMap[roleName], ",")
			for _, policy := range policies {
				deletePolicy = fmt.Sprintf("\taws iam delete-role-policy --role-name  %s  --policy-name  %s",
					roleName, policy)
				commands = append(commands, deletePolicy)
			}
		}
		deleteRole := fmt.Sprintf("\taws iam delete-role --role-name  %s", roleName)
		commands = append(commands, deleteRole)
	}
	return strings.Join(commands, "\n")
}
