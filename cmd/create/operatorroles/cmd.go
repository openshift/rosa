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
	clusterKey          string
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

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to create the roles for (required).",
	)
	Cmd.MarkFlagRequired("cluster")

	flags.StringVar(
		&args.prefix,
		"prefix",
		"",
		"Prefix to use for all IAM roles used by the operators needed in the OpenShift installer. "+
			"Leave empty to use an auto-generated one.",
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
		args.clusterKey = argv[0]
		args.mode = argv[1]
		args.permissionsBoundary = argv[2]

		if args.mode != "" {
			skipInteractive = true
		}
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
	operatorRolesPrefix := args.prefix
	if len(cluster.AWS().STS().OperatorIAMRoles()) > 0 {
		currentPrefix := getPrefix(cluster.AWS().STS().OperatorIAMRoles())
		//If the user provides in args we validate
		if operatorRolesPrefix != "" && currentPrefix != operatorRolesPrefix {
			reporter.Errorf("Cannot modify the existing prefix %s", operatorRolesPrefix)
			os.Exit(1)
		}
		operatorRolesPrefix = currentPrefix
	} else if operatorRolesPrefix == "" {
		operatorRolesPrefix = createRolePrefix(cluster.Name())
	}

	// Check to see if IAM operator roles have already created
	missingRoles, err := validateOperatorRoleWithRoleName(awsClient, operatorRolesPrefix)
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
	//We dont ask for the prefix if the user is attempting to create only the missing roles
	if interactive.Enabled() && len(cluster.AWS().STS().OperatorIAMRoles()) == 0 {
		operatorRolesPrefix, err = interactive.GetString(interactive.Input{
			Question: "Operator roles prefix",
			Help:     cmd.Flags().Lookup("prefix").Usage,
			Required: true,
			Default:  operatorRolesPrefix,
			Validators: []interactive.Validator{
				interactive.RegExp(aws.RoleNameRE.String()),
				interactive.MaxLength(32),
			},
		})
		if err != nil {
			reporter.Errorf("Expected a prefix for the operator IAM roles: %s", err)
			os.Exit(1)
		}
	}
	if len(operatorRolesPrefix) == 0 {
		reporter.Errorf("Expected a prefix for the operator IAM roles: %s", err)
		os.Exit(1)
	}
	if len(operatorRolesPrefix) > 32 {
		reporter.Errorf("Expected a prefix with no more than 32 characters")
		os.Exit(1)
	}
	if !aws.RoleNameRE.MatchString(operatorRolesPrefix) {
		reporter.Errorf("Expected valid operator roles prefix matching %s", aws.RoleNameRE.String())
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

	roleARN := cluster.AWS().STS().RoleARN()
	roleName := strings.Split(roleARN, "/")
	accountRolesPrefix := ""
	if len(roleName) > 1 {
		accountRolesPrefix = strings.Split(roleName[1], "-Installer-Role")[0]
	}

	switch mode {
	case "auto":
		ocmClient.LogEvent("ROSACreateOperatorRolesModeAuto")
		reporter.Infof("Creating roles using '%s'", creator.ARN)
		err = createRoles(reporter, awsClient, accountRolesPrefix, operatorRolesPrefix, permissionsBoundary,
			cluster, creator.AccountID)
		if err != nil {
			reporter.Errorf("There was an error creating the operator roles: %s", err)
			os.Exit(1)
		}
	case "manual":
		ocmClient.LogEvent("ROSACreateOperatorRolesModeManual")

		commands, err := buildCommands(reporter, accountRolesPrefix, operatorRolesPrefix, permissionsBoundary, cluster,
			creator.AccountID)
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

	operatorIAMRoleList := []ocm.OperatorIAMRole{}
	for _, operator := range aws.CredentialRequests {
		operatorIAMRoleList = append(operatorIAMRoleList, ocm.OperatorIAMRole{
			Name:      operator.Name,
			Namespace: operator.Namespace,
			RoleARN:   getOperatorRoleArn(operatorRolesPrefix, operator, creator),
		})
	}
	clusterConfig := ocm.Spec{
		OperatorIAMRoles: operatorIAMRoleList,
	}
	err = ocmClient.UpdateCluster(clusterKey, creator, clusterConfig)
	if err != nil {
		reporter.Errorf("Error updating the cluster with the operator roles %v", err)
	}
}

func getPrefix(operatorRoles []*cmv1.OperatorIAMRole) string {
	for _, operatorRole := range operatorRoles {
		for _, operator := range aws.CredentialRequests {
			if operatorRole.Name() == operator.Name {
				p := strings.Split(operatorRole.RoleARN(), fmt.Sprintf("-%s-%s", operator.Namespace, operator.Name))
				if len(p) > 0 {
					prefixArr := strings.Split(p[0], "/")
					if len(prefixArr) > 0 {
						return prefixArr[1]
					}
				}
			}
		}
	}
	return ""
}

func createRoles(reporter *rprtr.Object, awsClient aws.Client,
	accountRolesPrefix string, operatorRolesPrefix string, permissionsBoundary string,
	cluster *cmv1.Cluster, accountID string) error {
	version := getVersionMinor(cluster)

	for _, operator := range aws.CredentialRequests {
		roleName, err := getRoleName(operatorRolesPrefix, operator)
		if err != nil {
			return err
		}
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
		policyARN := getPolicyARN(accountID, accountRolesPrefix, operator.Namespace, operator.Name)
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
	accountRolesPrefix string, operatorRolesPrefix string, permissionsBoundary string,
	cluster *cmv1.Cluster, accountID string) (string, error) {
	commands := []string{}

	for credrequest, operator := range aws.CredentialRequests {
		roleName, err := getRoleName(operatorRolesPrefix, operator)
		if err != nil {
			return roleName, err
		}
		policyARN := getPolicyARN(accountID, accountRolesPrefix, operator.Namespace, operator.Name)
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
			tags.RolePrefix, accountRolesPrefix,
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

func getRoleName(operatorRolesPrefix string, operator aws.Operator) (string, error) {
	roleName := fmt.Sprintf("%s-%s-%s", operatorRolesPrefix, operator.Namespace, operator.Name)
	if len(roleName) > 64 {
		roleName = roleName[0:64]
	}
	if roleName == "" {
		return "", fmt.Errorf("Failed to find operator IAM role")
	}
	return roleName, nil
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

func validateOperatorRoleWithRoleName(awsClient aws.Client, prefix string) ([]string, error) {
	var missingRoles []string

	for _, operator := range aws.CredentialRequests {
		role := fmt.Sprintf("%s-%s-%s", prefix, operator.Namespace, operator.Name)
		if len(role) > 64 {
			role = role[0:64]
		}
		exists, err := awsClient.CheckRoleExists(role)
		if err != nil {
			return missingRoles, err
		}
		if !exists {
			missingRoles = append(missingRoles, role)
		}
	}
	return missingRoles, nil
}

func createRolePrefix(clusterName string) string {
	return fmt.Sprintf("%s-%s", clusterName, ocm.RandomLabel(4))
}

func getOperatorRoleArn(prefix string, operator aws.Operator, creator *aws.Creator) string {
	role := fmt.Sprintf("%s-%s-%s", prefix, operator.Namespace, operator.Name)
	if len(role) > 64 {
		role = role[0:64]
	}
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", creator.AccountID, role)
}
