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
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
	semver "github.com/hashicorp/go-version"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

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

	flags.StringVar(
		&args.permissionsBoundary,
		"permissions-boundary",
		"",
		"The ARN of the policy that is used to set the permissions boundary for the operator roles.",
	)

	flags.StringVar(
		&args.mode,
		"mode",
		modes[0],
		"How to perform the operation. Valid options are:\n"+
			"auto: Roles will be created using the current AWS account\n"+
			"manual: Role files will be saved in the current directory",
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

	// Allow the command to be called programmatically
	skipInteractive := false
	if len(argv) == 3 && !cmd.Flag("cluster").Changed {
		ocm.SetClusterKey(argv[0])
		args.mode = argv[1]
		args.permissionsBoundary = argv[2]

		if args.mode != "" {
			skipInteractive = true
		}
	}

	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Determine if interactive mode is needed
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") && !skipInteractive {
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

	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := ocmClient.GetCluster(clusterKey, creator)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.AWS().STS().RoleARN() == "" {
		reporter.Errorf("Cluster '%s' is not an STS cluster.", clusterKey)
		os.Exit(1)
	}

	// Check to see if IAM operator roles have already created
	missingRoles, err := validateOperatorRoles(awsClient, cluster)
	if err != nil {
		if strings.Contains(err.Error(), "AccessDenied") {
			reporter.Debugf("Failed to verify if operator roles exist: %s", err)
		} else {
			reporter.Errorf("Failed to verify if operator roles exist: %s", err)
			os.Exit(1)
		}
	}

	if len(missingRoles) == 0 &&
		cluster.State() != cmv1.ClusterStateWaiting && cluster.State() != cmv1.ClusterStatePending {
		reporter.Infof("Cluster '%s' is %s and does not need additional configuration.",
			clusterKey, cluster.State())
		os.Exit(0)
	}

	prefix := args.prefix
	if interactive.Enabled() {
		prefix, err = interactive.GetString(interactive.Input{
			Question: "Operator policy prefix",
			Help:     cmd.Flags().Lookup("prefix").Usage,
			Default:  prefix,
			Validators: []interactive.Validator{
				interactive.RegExp(aws.RoleNameRE.String()),
				interactive.MaxLength(32),
			},
		})
		if err != nil {
			reporter.Errorf("Expected a valid operator policy prefix: %s", err)
			os.Exit(1)
		}
	}
	if len(prefix) > 32 {
		reporter.Errorf("Expected a prefix with no more than 32 characters")
		os.Exit(1)
	}
	if prefix != "" && !aws.RoleNameRE.MatchString(prefix) {
		reporter.Errorf("Expected a valid operator policy prefix matching %s", aws.RoleNameRE.String())
		os.Exit(1)
	}
	// if prefix is empty, get prefix from cluster role ARN
	if prefix == "" {
		foundPrefix, exists := getPrefixFromAccountRole(cluster)
		if !exists {
			reporter.Errorf("Failed to find prefix from Account Role '%s'", aws.InstallerAccountRole)
			os.Exit(1)
		}
		prefix = foundPrefix
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
		ocmClient.LogEvent("ROSACreateOperatorRolesModeAuto")
		reporter.Infof("Creating roles using '%s'", creator.ARN)
		err = createRoles(reporter, awsClient, prefix, permissionsBoundary, cluster, creator.AccountID)
		if err != nil {
			reporter.Errorf("There was an error creating the operator roles: %s", err)
			os.Exit(1)
		}
	case "manual":
		ocmClient.LogEvent("ROSACreateOperatorRolesModeManual")

		commands, err := buildCommands(reporter, prefix, permissionsBoundary, cluster, creator.AccountID)
		if err != nil {
			reporter.Errorf("There was an error building the list of resources: %s", err)
			os.Exit(1)
		}

		if reporter.IsTerminal() {
			reporter.Infof("Run the following commands to create the operator roles:\n")
		}

		fmt.Println(commands)
	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", modes)
		os.Exit(1)
	}
}

func createRoles(reporter *rprtr.Object, awsClient aws.Client,
	prefix string, permissionsBoundary string,
	cluster *cmv1.Cluster, accountID string) error {
	version := getVersionMinor(cluster)

	for _, operator := range aws.CredentialRequests {
		roleName := getRoleName(cluster, operator)
		if roleName == "" {
			return fmt.Errorf("Failed to find operator IAM role")
		}

		if !confirm.Prompt(true, "Create the '%s' role?", roleName) {
			continue
		}

		policy, err := generateRolePolicyDoc(cluster, accountID, operator)
		if err != nil {
			return err
		}

		reporter.Debugf("Creating role '%s'", roleName)
		roleARN, err := awsClient.EnsureRole(roleName, policy, permissionsBoundary, version, map[string]string{
			tags.ClusterID:        cluster.ID(),
			tags.OpenShiftVersion: version,
			"operator_namespace":  operator.Namespace,
			"operator_name":       operator.Name,
		})
		if err != nil {
			return err
		}
		reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)

		policyARN := ""
		if prefix == "" {
			policyARN, err = awsClient.FindPolicyARN(operator, version)
			if err != nil {
				return err
			}
		}
		if policyARN == "" {
			policyARN = getPolicyARN(accountID, prefix, operator.Namespace, operator.Name)
		}

		reporter.Debugf("Attaching permission policy '%s' to role '%s'", policyARN, roleName)
		err = awsClient.AttachRolePolicy(roleName, policyARN)
		if err != nil {
			return fmt.Errorf("Failed to attach role policy. Check your prefix or run "+
				"'rosa create account-roles' to create the necessary policies: %s", err)
		}
	}

	return nil
}

func buildCommands(reporter *rprtr.Object,
	prefix string, permissionsBoundary string,
	cluster *cmv1.Cluster, accountID string) (string, error) {
	commands := []string{}

	for credrequest, operator := range aws.CredentialRequests {
		roleName := getRoleName(cluster, operator)
		policyARN := getPolicyARN(accountID, prefix, operator.Namespace, operator.Name)
		version := getVersionMinor(cluster)

		policy, err := generateRolePolicyDoc(cluster, accountID, operator)
		if err != nil {
			return "", err
		}

		filename := fmt.Sprintf("operator_%s_policy.json", credrequest)
		reporter.Debugf("Saving '%s' to the current directory", filename)
		err = saveDocument(policy, filename)
		if err != nil {
			return "", err
		}

		iamTags := fmt.Sprintf(
			"Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s",
			tags.ClusterID, cluster.ID(),
			tags.OpenShiftVersion, version,
			tags.RolePrefix, prefix,
			"operator_namespace", operator.Namespace,
			"operator_name", operator.Name,
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
		attachRolePolicy := fmt.Sprintf("aws iam attach-role-policy \\\n"+
			"\t--role-name %s \\\n"+
			"\t--policy-arn %s",
			roleName, policyARN)
		commands = append(commands, createRole, attachRolePolicy)
	}

	return strings.Join(commands, "\n\n"), nil
}

func generateRolePolicyDoc(cluster *cmv1.Cluster, accountID string, operator aws.Operator) (string, error) {
	version := getVersionMinor(cluster)

	oidcEndpointURL, err := url.ParseRequestURI(cluster.AWS().STS().OIDCEndpointURL())
	if err != nil {
		return "", err
	}
	issuerURL := fmt.Sprintf("%s%s", oidcEndpointURL.Host, oidcEndpointURL.Path)

	oidcProviderARN := fmt.Sprintf("arn:aws:iam::%s:oidc-provider/%s", accountID, issuerURL)

	serviceAccounts := []string{}
	for _, sa := range operator.ServiceAccountNames {
		serviceAccounts = append(serviceAccounts,
			fmt.Sprintf("system:serviceaccount:%s:%s", operator.Namespace, sa))
	}

	path := fmt.Sprintf("templates/policies/%s/operator_iam_role_policy.json", version)
	policy, err := aws.ReadPolicyDocument(path, map[string]string{
		"oidc_provider_arn": oidcProviderARN,
		"issuer_url":        issuerURL,
		"service_accounts":  strings.Join(serviceAccounts, `" , "`),
	})
	if err != nil {
		return "", err
	}

	return string(policy), nil
}

func saveDocument(doc string, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(doc)
	if err != nil {
		return err
	}

	return nil
}

func getRoleName(cluster *cmv1.Cluster, operator aws.Operator) string {
	for _, role := range cluster.AWS().STS().OperatorIAMRoles() {
		if role.Namespace() == operator.Namespace && role.Name() == operator.Name {
			return strings.SplitN(role.RoleARN(), "/", 2)[1]
		}
	}
	return ""
}

func getPolicyARN(accountID string, prefix string, namespace string, name string) string {
	if prefix == "" {
		prefix = aws.DefaultPrefix
	}
	policy := fmt.Sprintf("%s-%s-%s", prefix, namespace, name)
	if len(policy) > 64 {
		policy = policy[0:64]
	}
	return fmt.Sprintf("arn:aws:iam::%s:policy/%s", accountID, policy)
}

func getVersionMinor(cluster *cmv1.Cluster) string {
	// FIXME: OCM has a bug that prevents it from
	// returning the version Raw ID, so we extract it ourselves
	rawID := strings.Replace(cluster.Version().ID(), "openshift-v", "", 1)
	version, err := semver.NewVersion(rawID)
	if err != nil {
		segments := strings.Split(rawID, ".")
		return fmt.Sprintf("%s.%s", segments[0], segments[1])
	}
	segments := version.Segments()
	return fmt.Sprintf("%d.%d", segments[0], segments[1])
}

func validateOperatorRoles(awsClient aws.Client, cluster *cmv1.Cluster) ([]string, error) {
	var missingRoles []string

	operatorIAMRoles := cluster.AWS().STS().OperatorIAMRoles()

	if len(operatorIAMRoles) == 0 {
		return missingRoles, fmt.Errorf("No Operator IAM roles found for cluster %s", cluster.Name())
	}

	for _, operatorIAMRole := range operatorIAMRoles {
		roleARN := operatorIAMRole.RoleARN()

		roleName := strings.Split(roleARN, "/")[1]

		exists, err := awsClient.CheckRoleExists(roleName)
		if err != nil {
			return missingRoles, err
		}

		if !exists {
			missingRoles = append(missingRoles, roleName)
		}
	}

	return missingRoles, nil
}

func getPrefixFromAccountRole(cluster *cmv1.Cluster) (string, bool) {
	role, exists := aws.AccountRoles[aws.InstallerAccountRole]
	if !exists {
		return "", false
	}
	splitARN := strings.Split(cluster.AWS().STS().RoleARN(), "role/")
	if len(splitARN) != 2 {
		return "", false
	}
	splitRoleName := strings.Split(splitARN[1], fmt.Sprintf("-%s", role.Name))
	if len(splitRoleName) != 2 {
		return "", false
	}
	return splitRoleName[0], true
}
