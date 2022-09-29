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

package operatorroles

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	prefix              string
	permissionsBoundary string
}

var Cmd = &cobra.Command{
	Use:     "operator-roles",
	Aliases: []string{"operatorroles"},
	Short:   "Create operator IAM roles for a cluster.",
	Long:    "Create cluster-specific operator IAM roles based on your cluster configuration.",
	Example: `  # Create default operator roles for cluster named "mycluster"
  rosa create operator-roles --cluster=mycluster

  # Create operator roles with a specific permissions boundary
  rosa create operator-roles -c mycluster --permissions-boundary arn:aws:iam::123456789012:policy/perm-boundary`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)

	flags.StringVar(
		&args.prefix,
		"prefix",
		"",
		"User-defined prefix for generated AWS operator policies. Leave empty to attempt to find them automatically.",
	)
	flags.MarkDeprecated("prefix", "skip --prefix;rosa auto-detects prefix")

	flags.StringVar(
		&args.permissionsBoundary,
		"permissions-boundary",
		"",
		"The ARN of the policy that is used to set the permissions boundary for the operator roles.",
	)

	aws.AddModeFlag(Cmd)
	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	// Allow the command to be called programmatically
	skipInteractive := false
	if len(argv) == 3 && !cmd.Flag("cluster").Changed {
		ocm.SetClusterKey(argv[0])
		aws.SetModeKey(argv[1])
		args.permissionsBoundary = argv[2]

		// if mode is empty skip interactive is true
		if argv[1] != "" {
			skipInteractive = true
		}
	}

	clusterKey := r.GetClusterKey()

	mode, err := aws.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Determine if interactive mode is needed
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") && !skipInteractive {
		interactive.Enable()
	}

	cluster := r.FetchCluster()
	if cluster.AWS().STS().RoleARN() == "" {
		r.Reporter.Errorf("Cluster '%s' is not an STS cluster.", clusterKey)
		os.Exit(1)
	}

	// Check to see if IAM operator roles have already created
	missingRoles, err := validateOperatorRoles(r, cluster)
	if err != nil {
		if strings.Contains(err.Error(), "AccessDenied") {
			r.Reporter.Debugf("Failed to verify if operator roles exist: %s", err)
		} else {
			r.Reporter.Errorf("Failed to verify if operator roles exist: %s", err)
			os.Exit(1)
		}
	}

	if len(missingRoles) == 0 &&
		cluster.State() != cmv1.ClusterStateWaiting && cluster.State() != cmv1.ClusterStatePending {
		r.Reporter.Infof("Cluster '%s' is %s and does not need additional configuration.",
			clusterKey, cluster.State())
		os.Exit(0)
	}

	prefix, err := aws.GetPrefixFromAccountRole(cluster)
	if err != nil {
		r.Reporter.Errorf("Failed to find prefix from %s account role", aws.InstallerAccountRole)
		os.Exit(1)
	}

	permissionsBoundary := args.permissionsBoundary
	if interactive.Enabled() {
		permissionsBoundary, err = interactive.GetString(interactive.Input{
			Question: "Permissions boundary ARN",
			Help:     cmd.Flags().Lookup("permissions-boundary").Usage,
			Default:  permissionsBoundary,
			Validators: []interactive.Validator{
				aws.ARNValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid policy ARN for permissions boundary: %s", err)
			os.Exit(1)
		}
	}
	if permissionsBoundary != "" {
		_, err := arn.Parse(permissionsBoundary)
		if err != nil {
			r.Reporter.Errorf("Expected a valid policy ARN for permissions boundary: %s", err)
			os.Exit(1)
		}
	}

	if interactive.Enabled() && !skipInteractive {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "Role creation mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  aws.ModeAuto,
			Options:  aws.Modes,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid role creation mode: %s", err)
			os.Exit(1)
		}
	}

	roleName, err := aws.GetAccountRoleName(cluster)
	if err != nil {
		r.Reporter.Errorf("Expected parsing role account role '%s': %v", cluster.AWS().STS().RoleARN(), err)
		os.Exit(1)
	}
	path, err := getPathFromInstallerRole(cluster)
	if err != nil {
		r.Reporter.Errorf("Expected a valid path for  '%s': %v", cluster.AWS().STS().RoleARN(), err)
		os.Exit(1)
	}
	if path != "" {
		r.Reporter.Infof("Path '%s' detected, this path will be used for subsequent"+
			" created operator roles and policies.", path)
	}
	accountRoleVersion, err := r.AWSClient.GetAccountRoleVersion(roleName)
	if err != nil {
		r.Reporter.Errorf("Error getting account role version %s", err)
		os.Exit(1)
	}
	policies, err := r.OCMClient.GetPolicies("OperatorRole")
	if err != nil {
		r.Reporter.Errorf("Expected a valid role creation mode: %s", err)
		os.Exit(1)
	}

	env, err := ocm.GetEnv()
	if err != nil {
		r.Reporter.Errorf("Failed to determine OCM environment: %v", err)
		os.Exit(1)
	}

	defaultPolicyVersion, err := r.OCMClient.GetDefaultVersion()
	if err != nil {
		r.Reporter.Errorf("Error getting latest default version: %s", err)
		os.Exit(1)
	}

	credRequests, err := r.OCMClient.GetCredRequests(cluster.Hypershift().Enabled())
	if err != nil {
		r.Reporter.Errorf("Error getting operator credential request from OCM %s", err)
		os.Exit(1)
	}

	switch mode {
	case aws.ModeAuto:
		if !output.HasFlag() || r.Reporter.IsTerminal() {
			r.Reporter.Infof("Creating roles using '%s'", r.Creator.ARN)
		}
		err = createRoles(r, prefix, permissionsBoundary, cluster,
			accountRoleVersion, policies, defaultPolicyVersion, credRequests)
		if err != nil {
			r.Reporter.Errorf("There was an error creating the operator roles: %s", err)
			isThrottle := "false"
			if strings.Contains(err.Error(), "Throttling") {
				isThrottle = "true"
			}
			r.OCMClient.LogEvent("ROSACreateOperatorRolesModeAuto", map[string]string{
				ocm.ClusterID:  clusterKey,
				ocm.Response:   ocm.Failure,
				ocm.IsThrottle: isThrottle,
			})
			os.Exit(1)
		}
		r.OCMClient.LogEvent("ROSACreateOperatorRolesModeAuto", map[string]string{
			ocm.ClusterID: clusterKey,
			ocm.Response:  ocm.Success,
		})
	case aws.ModeManual:
		commands, err := buildCommands(r, env, prefix, permissionsBoundary, defaultPolicyVersion,
			cluster, policies, credRequests)
		if err != nil {
			r.Reporter.Errorf("There was an error building the list of resources: %s", err)
			os.Exit(1)
			r.OCMClient.LogEvent("ROSACreateOperatorRolesModeManual", map[string]string{
				ocm.ClusterID: clusterKey,
				ocm.Response:  ocm.Failure,
			})
		}
		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("All policy files saved to the current directory")
			r.Reporter.Infof("Run the following commands to create the operator roles:\n")
		}
		r.OCMClient.LogEvent("ROSACreateOperatorRolesModeManual", map[string]string{
			ocm.ClusterID: clusterKey,
		})
		fmt.Println(commands)

	default:
		r.Reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
}

func createRoles(r *rosa.Runtime,
	prefix string, permissionsBoundary string,
	cluster *cmv1.Cluster, accountRoleVersion string, policies map[string]string,
	defaultVersion string, credRequests map[string]*cmv1.STSOperator) error {
	for credrequest, operator := range credRequests {
		ver := cluster.Version()
		if ver != nil && operator.MinVersion() != "" {
			isSupported, err := ocm.CheckSupportedVersion(ocm.GetVersionMinor(ver.ID()), operator.MinVersion())
			if err != nil {
				r.Reporter.Errorf("Error validating operator role '%s' version %s", operator.Name(), err)
				os.Exit(1)
			}
			if !isSupported {
				continue
			}
		}
		roleName, _ := getRoleNameAndARN(cluster, operator)
		if roleName == "" {
			return fmt.Errorf("Failed to find operator IAM role")
		}
		if !confirm.Prompt(true, "Create the '%s' role?", roleName) {
			continue
		}

		path, err := getPathFromInstallerRole(cluster)
		if err != nil {
			return err
		}
		policyARN := aws.GetOperatorPolicyARN(r.Creator.AccountID, prefix, operator.Namespace(),
			operator.Name(), path)
		filename := fmt.Sprintf("openshift_%s_policy", credrequest)
		policyDetails := policies[filename]

		policyARN, err = r.AWSClient.EnsurePolicy(policyARN, policyDetails,
			defaultVersion, map[string]string{
				tags.OpenShiftVersion: accountRoleVersion,
				tags.RolePrefix:       prefix,
				tags.RedHatManaged:    "true",
				"operator_namespace":  operator.Namespace(),
				"operator_name":       operator.Name(),
			}, path)
		if err != nil {
			return err
		}
		policyDetails = policies["operator_iam_role_policy"]

		policy, err := aws.GenerateOperatorRolePolicyDoc(cluster, r.Creator.AccountID, operator, policyDetails)
		if err != nil {
			return err
		}
		r.Reporter.Debugf("Creating role '%s'", roleName)

		roleARN, err := r.AWSClient.EnsureRole(roleName, policy, permissionsBoundary, accountRoleVersion,
			map[string]string{
				tags.ClusterID:       cluster.ID(),
				"operator_namespace": operator.Namespace(),
				"operator_name":      operator.Name(),
				tags.RedHatManaged:   "true",
			}, path)
		if err != nil {
			return err
		}
		if !output.HasFlag() || r.Reporter.IsTerminal() {
			r.Reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)
		}

		r.Reporter.Debugf("Attaching permission policy '%s' to role '%s'", policyARN, roleName)
		err = r.AWSClient.AttachRolePolicy(roleName, policyARN)
		if err != nil {
			return err
		}
	}

	return nil
}

func buildCommands(r *rosa.Runtime, env string,
	prefix string, permissionsBoundary string, defaultPolicyVersion string, cluster *cmv1.Cluster,
	policies map[string]string, credRequests map[string]*cmv1.STSOperator) (string, error) {

	err := aws.GeneratePolicyFiles(r.Reporter, env, false,
		true, policies, credRequests)
	if err != nil {
		r.Reporter.Errorf("There was an error generating the policy files: %s", err)
		os.Exit(1)
	}

	commands := []string{}

	for credrequest, operator := range credRequests {
		ver := cluster.Version()
		if ver != nil && operator.MinVersion() != "" {
			isSupported, err := ocm.CheckSupportedVersion(ocm.GetVersionMinor(ver.ID()), operator.MinVersion())
			if err != nil {
				r.Reporter.Errorf("Error validating operator role '%s' version %s", operator.Name(), err)
				os.Exit(1)
			}
			if !isSupported {
				continue
			}
		}
		roleName, _ := getRoleNameAndARN(cluster, operator)
		path, err := getPathFromInstallerRole(cluster)
		if err != nil {
			return "", err
		}
		policyARN := getPolicyARN(r.Creator.AccountID, prefix, operator.Namespace(), operator.Name(), path)

		name := aws.GetPolicyName(prefix, operator.Namespace(), operator.Name())
		_, err = r.AWSClient.IsPolicyExists(policyARN)
		if err != nil {
			iamTags := fmt.Sprintf(
				"Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s",
				tags.OpenShiftVersion, defaultPolicyVersion,
				tags.RolePrefix, prefix,
				"operator_namespace", operator.Namespace(),
				"operator_name", operator.Name(),
				tags.RedHatManaged, "true",
			)
			createPolicy := fmt.Sprintf("aws iam create-policy \\\n"+
				"\t--policy-name %s \\\n"+
				"\t--policy-document file://openshift_%s_policy.json \\\n"+
				"\t--tags %s",
				name, credrequest, iamTags)
			if path != "" {
				createPolicy = fmt.Sprintf(createPolicy+"\t--path %s", path)
			}
			commands = append(commands, createPolicy)
		}

		policyDetail := policies["operator_iam_role_policy"]
		policy, err := aws.GenerateOperatorRolePolicyDoc(cluster, r.Creator.AccountID, operator, policyDetail)
		if err != nil {
			return "", err
		}

		filename := fmt.Sprintf("operator_%s_policy", credrequest)
		filename = aws.GetFormattedFileName(filename)
		r.Reporter.Debugf("Saving '%s' to the current directory", filename)
		err = helper.SaveDocument(policy, filename)
		if err != nil {
			return "", err
		}
		iamTags := fmt.Sprintf(
			"Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s",
			tags.ClusterID, cluster.ID(),
			tags.RolePrefix, prefix,
			"operator_namespace", operator.Namespace(),
			"operator_name", operator.Name(),
			tags.RedHatManaged, "true",
		)
		permBoundaryFlag := ""
		if permissionsBoundary != "" {
			permBoundaryFlag = fmt.Sprintf("\t--permissions-boundary %s \\\n", permissionsBoundary)

		}
		createRole := fmt.Sprintf("aws iam create-role \\\n"+
			"\t--role-name %s \\\n"+
			"\t--assume-role-policy-document file://%s \\\n"+
			"%s"+
			"\t--tags %s",
			roleName, filename, permBoundaryFlag, iamTags)
		if path != "" {
			createRole = fmt.Sprintf(createRole+"\t--path %s", path)
		}
		attachRolePolicy := fmt.Sprintf("aws iam attach-role-policy \\\n"+
			"\t--role-name %s \\\n"+
			"\t--policy-arn %s",
			roleName, policyARN)
		commands = append(commands, createRole, attachRolePolicy)
	}
	return strings.Join(commands, "\n\n"), nil
}

func getRoleNameAndARN(cluster *cmv1.Cluster, operator *cmv1.STSOperator) (string, string) {
	for _, role := range cluster.AWS().STS().OperatorIAMRoles() {
		if role.Namespace() == operator.Namespace() && role.Name() == operator.Name() {
			name, _ := aws.GetResourceIdFromARN(role.RoleARN())
			return name, role.RoleARN()
		}
	}
	return "", ""
}

func getPathFromInstallerRole(cluster *cmv1.Cluster) (string, error) {
	return aws.GetPathFromARN(cluster.AWS().STS().RoleARN())
}

func getPolicyARN(accountID string, prefix string, namespace string, name string, path string) string {
	if prefix == "" {
		prefix = aws.DefaultPrefix
	}
	policy := fmt.Sprintf("%s-%s-%s", prefix, namespace, name)
	if len(policy) > 64 {
		policy = policy[0:64]
	}
	if path != "" {
		return fmt.Sprintf("arn:aws:iam::%s:policy%s%s", accountID, path, policy)
	}
	return fmt.Sprintf("arn:aws:iam::%s:policy/%s", accountID, policy)
}

func validateOperatorRoles(r *rosa.Runtime, cluster *cmv1.Cluster) ([]string, error) {
	var missingRoles []string
	operatorIAMRoles := cluster.AWS().STS().OperatorIAMRoles()
	if len(operatorIAMRoles) == 0 {
		return missingRoles, fmt.Errorf("No Operator IAM roles found for cluster %s", cluster.Name())
	}
	for _, operatorIAMRole := range operatorIAMRoles {
		roleARN := operatorIAMRole.RoleARN()
		roleName, err := aws.GetResourceIdFromARN(roleARN)
		if err != nil {
			return missingRoles, err
		}
		exists, _, err := r.AWSClient.CheckRoleExists(roleName)
		if err != nil {
			return missingRoles, err
		}
		if !exists {
			missingRoles = append(missingRoles, roleName)
		}
	}
	return missingRoles, nil
}
