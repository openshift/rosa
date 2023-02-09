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

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
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
	forcePolicyCreation bool
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

	flags.BoolVarP(
		&args.forcePolicyCreation,
		"force-policy-creation",
		"f",
		false,
		"Forces creation of policies skipping compatibility check",
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
		cluster.State() != cmv1.ClusterStateWaiting && cluster.State() != cmv1.ClusterStatePending &&
		!args.forcePolicyCreation {
		r.Reporter.Infof("Cluster '%s' is %s and does not need additional configuration.",
			clusterKey, cluster.State())
		os.Exit(0)
	}

	if args.forcePolicyCreation && mode != aws.ModeAuto {
		r.Reporter.Warnf("Forcing creation of policies only works in auto mode")
		os.Exit(1)
	}

	permissionsBoundary := args.permissionsBoundary
	if interactive.Enabled() && !skipInteractive {
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
		err = aws.ARNValidator(permissionsBoundary)
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

	roleName, err := aws.GetInstallerAccountRoleName(cluster)
	if err != nil {
		r.Reporter.Errorf("Expected parsing role account role '%s': %v", cluster.AWS().STS().RoleARN(), err)
		os.Exit(1)
	}

	path, err := getPathFromInstallerRole(cluster)
	if err != nil {
		r.Reporter.Errorf("Expected a valid path for '%s': %v", cluster.AWS().STS().RoleARN(), err)
		os.Exit(1)
	}
	if path != "" && !output.HasFlag() && r.Reporter.IsTerminal() {
		r.Reporter.Infof("ARN path '%s' detected in installer role '%s'. "+
			"This ARN path will be used for subsequent created operator roles and policies.",
			path, cluster.AWS().STS().RoleARN())
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

	managedPolicies, err := r.AWSClient.HasManagedPolicies(cluster.AWS().STS().RoleARN())
	if err != nil {
		r.Reporter.Errorf("Failed to determine if cluster has managed policies: %v", err)
		os.Exit(1)
	}
	if args.forcePolicyCreation && managedPolicies {
		r.Reporter.Warnf("Forcing creation of policies only works for unmanaged policies")
		os.Exit(1)
	}
	// TODO: remove once AWS managed policies are in place
	if managedPolicies && env == ocm.Production {
		r.Reporter.Errorf("Managed policies are not supported in this environment")
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

	operatorRolePolicyPrefix, err := aws.GetOperatorRolePolicyPrefixFromCluster(cluster, r.AWSClient)
	if err != nil {
		r.Reporter.Errorf("%s", err)
	}

	switch mode {
	case aws.ModeAuto:
		if !output.HasFlag() || r.Reporter.IsTerminal() {
			r.Reporter.Infof("Creating roles using '%s'", r.Creator.ARN)
		}
		err = createRoles(r, operatorRolePolicyPrefix, permissionsBoundary, cluster,
			accountRoleVersion, policies, defaultPolicyVersion, credRequests, managedPolicies)
		if err != nil {
			r.Reporter.Errorf("There was an error creating the operator roles: %s", err)
			isThrottle := "false"
			if strings.Contains(err.Error(), "Throttling") {
				isThrottle = helper.True
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
		commands, err := buildCommands(r, env, operatorRolePolicyPrefix, permissionsBoundary, defaultPolicyVersion,
			cluster, policies, credRequests, managedPolicies)
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
	cluster *cmv1.Cluster, accountRoleVersion string, policies map[string]*cmv1.AWSSTSPolicy,
	defaultVersion string, credRequests map[string]*cmv1.STSOperator, managedPolicies bool) error {
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
		roleName, _ := aws.FindOperatorRoleNameBySTSOperator(cluster, operator)
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

		var policyARN string
		filename := fmt.Sprintf("openshift_%s_policy", credrequest)
		if managedPolicies {
			policyARN, err = aws.GetManagedPolicyARN(policies, filename)
			if err != nil {
				return err
			}
		} else {
			policyARN = aws.GetOperatorPolicyARN(r.Creator.AccountID, prefix, operator.Namespace(),
				operator.Name(), path)
			policyDetails := aws.GetPolicyDetails(policies, filename)

			operatorPolicyTags := map[string]string{
				tags.OpenShiftVersion:  accountRoleVersion,
				tags.RolePrefix:        prefix,
				tags.RedHatManaged:     helper.True,
				tags.OperatorNamespace: operator.Namespace(),
				tags.OperatorName:      operator.Name(),
			}

			if args.forcePolicyCreation {
				policyARN, err = r.AWSClient.ForceEnsurePolicy(policyARN, policyDetails,
					defaultVersion, operatorPolicyTags, path)
				if err != nil {
					return err
				}
			} else {
				policyARN, err = r.AWSClient.EnsurePolicy(policyARN, policyDetails,
					defaultVersion, operatorPolicyTags, path)
				if err != nil {
					return err
				}
			}
		}

		policyDetails := aws.GetPolicyDetails(policies, "operator_iam_role_policy")
		policy, err := aws.GenerateOperatorRolePolicyDoc(cluster, r.Creator.AccountID, operator, policyDetails)
		if err != nil {
			return err
		}

		r.Reporter.Debugf("Creating role '%s'", roleName)
		tagsList := map[string]string{
			tags.ClusterID:         cluster.ID(),
			tags.OperatorNamespace: operator.Namespace(),
			tags.OperatorName:      operator.Name(),
			tags.RedHatManaged:     helper.True,
		}
		if managedPolicies {
			tagsList[tags.ManagedPolicies] = helper.True
		}

		roleARN, err := r.AWSClient.EnsureRole(roleName, policy, permissionsBoundary, accountRoleVersion,
			tagsList, path, managedPolicies)
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
	policies map[string]*cmv1.AWSSTSPolicy, credRequests map[string]*cmv1.STSOperator,
	managedPolicies bool) (string, error) {
	err := aws.GeneratePolicyFiles(r.Reporter, env, false,
		true, policies, credRequests, managedPolicies)
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
		roleName, _ := aws.FindOperatorRoleNameBySTSOperator(cluster, operator)
		path, err := getPathFromInstallerRole(cluster)
		if err != nil {
			return "", err
		}

		var policyARN string
		if managedPolicies {
			policyARN, err = aws.GetManagedPolicyARN(policies, fmt.Sprintf("openshift_%s_policy", credrequest))
			if err != nil {
				return "", err
			}
		} else {
			policyARN = getPolicyARN(r.Creator.AccountID, prefix, operator.Namespace(), operator.Name(), path)
			name := aws.GetOperatorPolicyName(prefix, operator.Namespace(), operator.Name())
			_, err = r.AWSClient.IsPolicyExists(policyARN)
			if err != nil {
				iamTags := map[string]string{
					tags.OpenShiftVersion:  defaultPolicyVersion,
					tags.RolePrefix:        prefix,
					tags.OperatorNamespace: operator.Namespace(),
					tags.OperatorName:      operator.Name(),
					tags.RedHatManaged:     helper.True,
				}
				createPolicy := awscb.NewIAMCommandBuilder().
					SetCommand(awscb.CreatePolicy).
					AddParam(awscb.PolicyName, name).
					AddParam(awscb.PolicyDocument, fmt.Sprintf("file://openshift_%s_policy.json", credrequest)).
					AddTags(iamTags).
					AddParam(awscb.Path, path).
					Build()
				commands = append(commands, createPolicy)
			}
		}

		policyDetail := aws.GetPolicyDetails(policies, "operator_iam_role_policy")
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
		iamTags := map[string]string{
			tags.ClusterID:         cluster.ID(),
			tags.RolePrefix:        prefix,
			tags.OperatorNamespace: operator.Namespace(),
			tags.OperatorName:      operator.Name(),
			tags.RedHatManaged:     helper.True,
		}
		if managedPolicies {
			iamTags[tags.ManagedPolicies] = helper.True
		}
		createRole := awscb.NewIAMCommandBuilder().
			SetCommand(awscb.CreateRole).
			AddParam(awscb.RoleName, roleName).
			AddParam(awscb.AssumeRolePolicyDocument, fmt.Sprintf("file://%s", filename)).
			AddParam(awscb.PermissionsBoundary, permissionsBoundary).
			AddTags(iamTags).
			AddParam(awscb.Path, path).
			Build()

		attachRolePolicy := awscb.NewIAMCommandBuilder().
			SetCommand(awscb.AttachRolePolicy).
			AddParam(awscb.RoleName, roleName).
			AddParam(awscb.PolicyArn, policyARN).
			Build()
		commands = append(commands, createRole, attachRolePolicy)
	}
	return awscb.JoinCommands(commands), nil
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
		return fmt.Sprintf("arn:%s:iam::%s:policy%s%s", aws.GetPartition(), accountID, path, policy)
	}
	return fmt.Sprintf("arn:%s:iam::%s:policy/%s", aws.GetPartition(), accountID, policy)
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
