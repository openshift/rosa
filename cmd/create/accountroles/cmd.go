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

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/login"
	"github.com/openshift/rosa/cmd/verify/oc"
	"github.com/openshift/rosa/cmd/verify/quota"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var modes []string = []string{"auto", "manual"}

var args struct {
	prefix              string
	permissionsBoundary string
	mode                string
}

var Cmd = &cobra.Command{
	Use:     "account-roles",
	Aliases: []string{"accountroles", "roles", "policies"},
	Short:   "Create account-wide IAM roles before creating your cluster.",
	Long:    "Create account-wide IAM roles before creating your cluster.",
	Example: `  # Create default account roles for ROSA clusters using STS
  rosa create account-roles

  # Create account roles with a specific permissions boundary
  rosa create account-roles --permissions-boundary arn:aws:iam::123456789012:policy/perm-boundary`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.prefix,
		"prefix",
		aws.DefaultPrefix,
		"User-defined prefix for all generated AWS resources",
	)

	flags.StringVar(
		&args.permissionsBoundary,
		"permissions-boundary",
		"",
		"The ARN of the policy that is used to set the permissions boundary for the account roles.",
	)

	flags.StringVar(
		&args.mode,
		"mode",
		modes[0],
		"How to perform the operation. Valid options are:\n"+
			"auto: Roles and policies will be created using the current AWS account\n"+
			"manual: Policy documents will be saved in the current directory",
	)
	Cmd.RegisterFlagCompletionFunc("mode", modeCompletion)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func modeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return modes, cobra.ShellCompDirectiveDefault
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// If necessary, call `login` as part of `init`. We do this before
	// other validations to get the prompt out of the way before performing
	// longer checks.
	err := login.Call(cmd, argv, reporter)
	if err != nil {
		reporter.Errorf("Failed to login to OCM: %v", err)
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
		reporter.Errorf("Failed to determine OCM environment: %v", err)
		os.Exit(1)
	}

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create AWS client: %v", err)
		os.Exit(1)
	}

	// Validate AWS credentials for current user
	if reporter.IsTerminal() {
		reporter.Infof("Validating AWS credentials...")
	}
	ok, err := awsClient.ValidateCredentials()
	if err != nil {
		ocmClient.LogEvent("ROSAInitCredentialsFailed")
		reporter.Errorf("Error validating AWS credentials: %v", err)
		os.Exit(1)
	}
	if !ok {
		ocmClient.LogEvent("ROSAInitCredentialsInvalid")
		reporter.Errorf("AWS credentials are invalid")
		os.Exit(1)
	}
	if reporter.IsTerminal() {
		reporter.Infof("AWS credentials are valid!")
	}

	creator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Unable to get IAM credentials: %s", err)
		os.Exit(1)
	}

	// Validate AWS quota
	// Call `verify quota` as part of init
	err = quota.Cmd.RunE(cmd, argv)
	if err != nil {
		reporter.Warnf("Insufficient AWS quotas. Cluster installation might fail.")
	}
	// Verify version of `oc`
	oc.Cmd.Run(cmd, argv)

	// Determine if interactive mode is needed
	if !interactive.Enabled() && (!cmd.Flags().Changed("mode")) {
		interactive.Enable()
	}

	if reporter.IsTerminal() {
		reporter.Infof("Creating account roles")
	}
	reporter.Debugf("Creating account roles compatible with OpenShift versions up to %s", aws.DefaultPolicyVersion)

	prefix := args.prefix
	if interactive.Enabled() {
		prefix, err = interactive.GetString(interactive.Input{
			Question: "Role prefix",
			Help:     cmd.Flags().Lookup("prefix").Usage,
			Default:  prefix,
			Required: true,
			Validators: []interactive.Validator{
				interactive.RegExp(`[\w+=,.@-]+`),
				interactive.MaxLength(32),
			},
		})
		if err != nil {
			reporter.Errorf("Expected a valid role prefix: %s", err)
			os.Exit(1)
		}
	}
	if len(prefix) > 32 {
		reporter.Errorf("Expected a prefix with no more than 32 characters")
		os.Exit(1)
	}
	if !aws.RoleNameRE.MatchString(prefix) {
		reporter.Errorf("Expected a valid role prefix matching %s", aws.RoleNameRE.String())
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
			reporter.Errorf("Expected a valid policy ARN for permissions boundary: %s", err)
			os.Exit(1)
		}
	}
	if permissionsBoundary != "" {
		_, err := arn.Parse(permissionsBoundary)
		if err != nil {
			reporter.Errorf("Expected a valid policy ARN for permissions boundary: %s", err)
			os.Exit(1)
		}
	}

	mode := args.mode
	if interactive.Enabled() {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "Role creation mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  mode,
			Options:  modes,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid role creation mode: %s", err)
			os.Exit(1)
		}
	}

	switch mode {
	case "auto":
		ocmClient.LogEvent("ROSACreateAccountRolesModeAuto")
		reporter.Infof("Creating roles using '%s'", creator.ARN)
		err = createRoles(reporter, awsClient, prefix, permissionsBoundary, creator.AccountID, env)
		if err != nil {
			reporter.Errorf("There was an error creating the account roles: %s", err)
			os.Exit(1)
		}
		reporter.Infof("To create a cluster with these roles, run the following command:\n" +
			"rosa create cluster --sts")
	case "manual":
		ocmClient.LogEvent("ROSACreateAccountRolesModeManual")
		err = generatePolicyFiles(reporter, env)
		if err != nil {
			reporter.Errorf("There was an error generating the policy files: %s", err)
			os.Exit(1)
		}

		if reporter.IsTerminal() {
			reporter.Infof("All policy files saved to the current directory")
			reporter.Infof("Run the following commands to create the account roles and policies:\n")
		}

		commands := buildCommands(prefix, permissionsBoundary)
		fmt.Println(commands)
	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", modes)
		os.Exit(1)
	}
}

func generatePolicyFiles(reporter *rprtr.Object, env string) error {
	for file := range aws.AccountRoles {
		filename := fmt.Sprintf("sts_%s_trust_policy.json", file)
		path := fmt.Sprintf("templates/policies/%s", filename)

		policy, err := aws.ReadPolicyDocument(path, map[string]string{
			"aws_account_id": aws.JumpAccounts[env],
		})
		if err != nil {
			return err
		}

		reporter.Debugf("Saving '%s' to the current directory", filename)
		err = saveDocument(policy, filename)
		if err != nil {
			return err
		}

		filename = fmt.Sprintf("sts_%s_permission_policy.json", file)
		path = fmt.Sprintf("templates/policies/%s", filename)

		policy, err = aws.ReadPolicyDocument(path)
		if err != nil {
			return err
		}

		reporter.Debugf("Saving '%s' to the current directory", filename)
		err = saveDocument(policy, filename)
		if err != nil {
			return err
		}
	}

	for credrequest := range aws.CredentialRequests {
		filename := fmt.Sprintf("openshift_%s_policy.json", credrequest)
		path := fmt.Sprintf("templates/policies/%s", filename)

		policy, err := aws.ReadPolicyDocument(path)
		if err != nil {
			return err
		}

		reporter.Debugf("Saving '%s' to the current directory", filename)
		err = saveDocument(policy, filename)
		if err != nil {
			return err
		}
	}

	return nil
}

func saveDocument(doc []byte, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(doc)
	if err != nil {
		return err
	}

	return nil
}

func buildCommands(prefix string, permissionsBoundary string) string {
	commands := []string{}

	for file, role := range aws.AccountRoles {
		name := getRoleName(prefix, role.Name)
		iamTags := fmt.Sprintf(
			"Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s",
			tags.OpenShiftVersion, aws.DefaultPolicyVersion,
			tags.RolePrefix, prefix,
			tags.RoleType, file,
		)
		permBoundaryFlag := ""
		if permissionsBoundary != "" {
			permBoundaryFlag = fmt.Sprintf("\t--permissions-boundary %s \\\n", permissionsBoundary)
		}
		createRole := fmt.Sprintf("aws iam create-role \\\n"+
			"\t--role-name %s \\\n"+
			"\t--assume-role-policy-document file://sts_%s_trust_policy.json \\\n"+
			"%s"+
			"\t--tags %s",
			name, file, permBoundaryFlag, iamTags)
		putRolePolicy := fmt.Sprintf("aws iam put-role-policy \\\n"+
			"\t--role-name %s \\\n"+
			"\t--policy-name %s-Policy \\\n"+
			"\t--policy-document file://sts_%s_permission_policy.json",
			name, name, file)
		commands = append(commands, createRole, putRolePolicy)
	}

	for credrequest, operator := range aws.CredentialRequests {
		name := getPolicyName(prefix, operator.Namespace, operator.Name)
		iamTags := fmt.Sprintf(
			"Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s",
			tags.OpenShiftVersion, aws.DefaultPolicyVersion,
			tags.RolePrefix, prefix,
			"operator_namespace", operator.Namespace,
			"operator_name", operator.Name,
		)
		createPolicy := fmt.Sprintf("aws iam create-policy \\\n"+
			"\t--policy-name %s \\\n"+
			"\t--policy-document file://openshift_%s_policy.json \\\n"+
			"\t--tags %s",
			name, credrequest, iamTags)
		commands = append(commands, createPolicy)
	}

	return strings.Join(commands, "\n\n")
}

func createRoles(reporter *rprtr.Object, awsClient aws.Client,
	prefix string, permissionsBoundary string,
	accountID string, env string) error {
	for file, role := range aws.AccountRoles {
		name := getRoleName(prefix, role.Name)

		if !confirm.Prompt(true, "Create the '%s' role?", name) {
			continue
		}

		filename := fmt.Sprintf("sts_%s_trust_policy.json", file)
		path := fmt.Sprintf("templates/policies/%s", filename)

		policy, err := aws.ReadPolicyDocument(path, map[string]string{
			"aws_account_id": aws.JumpAccounts[env],
		})
		if err != nil {
			return err
		}

		roleARN, err := awsClient.EnsureRole(name, string(policy), permissionsBoundary,
			aws.DefaultPolicyVersion, map[string]string{
				tags.OpenShiftVersion: aws.DefaultPolicyVersion,
				tags.RolePrefix:       prefix,
				tags.RoleType:         file,
			})
		if err != nil {
			return err
		}
		reporter.Infof("Created role '%s' with ARN '%s'", name, roleARN)

		filename = fmt.Sprintf("sts_%s_permission_policy.json", file)
		path = fmt.Sprintf("templates/policies/%s", filename)

		policy, err = aws.ReadPolicyDocument(path)
		if err != nil {
			return err
		}

		reporter.Debugf("Attaching permission policy to role '%s'", filename)
		err = awsClient.PutRolePolicy(name, fmt.Sprintf("%s-Policy", name), string(policy))
		if err != nil {
			return err
		}
	}

	if confirm.Prompt(true, "Create the operator policies?") {
		for credrequest, operator := range aws.CredentialRequests {
			policyArn := getPolicyARN(accountID, prefix, operator.Namespace, operator.Name)

			filename := fmt.Sprintf("openshift_%s_policy.json", credrequest)
			path := fmt.Sprintf("templates/policies/%s", filename)

			policy, err := aws.ReadPolicyDocument(path)
			if err != nil {
				return err
			}

			policyArn, err = awsClient.EnsurePolicy(policyArn, string(policy),
				aws.DefaultPolicyVersion, map[string]string{
					tags.OpenShiftVersion: aws.DefaultPolicyVersion,
					tags.RolePrefix:       prefix,
					"operator_namespace":  operator.Namespace,
					"operator_name":       operator.Name,
				})
			if err != nil {
				return err
			}
			reporter.Infof("Created policy with ARN '%s'", policyArn)
		}
	}

	return nil
}

func getRoleName(prefix string, role string) string {
	name := fmt.Sprintf("%s-%s-Role", prefix, role)
	if len(name) > 64 {
		name = name[0:64]
	}
	return name
}

func getPolicyName(prefix string, namespace string, name string) string {
	policy := fmt.Sprintf("%s-%s-%s", prefix, namespace, name)
	if len(policy) > 64 {
		policy = policy[0:64]
	}
	return policy
}

func getPolicyARN(accountID string, prefix string, namespace string, name string) string {
	return fmt.Sprintf("arn:aws:iam::%s:policy/%s", accountID, getPolicyName(prefix, namespace, name))
}
