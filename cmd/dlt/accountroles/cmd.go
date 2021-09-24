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
	clusterKey string
	mode       string
}

var Cmd = &cobra.Command{
	Use:     "account-roles",
	Aliases: []string{"accountrole,account-role"},
	Short:   "Delete Account Roles",
	Long:    "Cleans up account roles from the current aws account.",
	Example: `  # Delete Account roles"
  rosa delete account-roles`,
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

	mode := args.mode
	if interactive.Enabled() {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "Account roles deletion mode",
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

	env, err := ocm.GetEnv()
	if err != nil {
		reporter.Errorf("Error getting environment %s", err)
		os.Exit(1)
	}

	roles, err := awsClient.GetAccountRolesForCurrentEnv(env)
	if err != nil {
		reporter.Errorf("Error getting roles %s", err)
		os.Exit(1)
	}
	reporter.Infof("%v", roles)

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
