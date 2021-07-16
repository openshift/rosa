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
	clusterKey string
	prefix     string
	mode       string
}

var Cmd = &cobra.Command{
	Use:     "operator-roles",
	Aliases: []string{"operatorroles"},
	Short:   "Create operator IAM roles for a cluster.",
	Long:    "Create cluster-specific operator IAM roles based on your cluster configuration.",
	Example: `  # Create default operator roles for cluster named "mycluster"
  rosa create operator-roles --cluster=mycluster`,
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
		"ManagedOpenShift",
		"User-defined prefix for generated AWS roles. Leave empty to use the cluster name.",
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

	if cluster.State() != cmv1.ClusterStatePending {
		reporter.Errorf("Cluster '%s' is %s and does not need additional configuration.",
			clusterKey, cluster.State())
		os.Exit(1)
	}

	if cluster.AWS().STS().RoleARN() == "" {
		reporter.Errorf("Cluster '%s' is not an STS cluster.", clusterKey)
		os.Exit(1)
	}

	prefix := args.prefix
	if interactive.Enabled() {
		prefix, err = interactive.GetString(interactive.Input{
			Question: "Role prefix",
			Help:     cmd.Flags().Lookup("prefix").Usage,
			Default:  prefix,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid role prefix: %s", err)
			os.Exit(1)
		}
	}
	if prefix == "" {
		prefix = cluster.Name()
	}
	if len(prefix) > 32 {
		reporter.Errorf("Expected a prefix with no more than 32 characters")
		os.Exit(1)
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
		reporter.Infof("Creating roles using '%s'", creator.ARN)
		err = createRoles(reporter, awsClient, prefix, cluster, creator.AccountID)
		if err != nil {
			reporter.Errorf("There was an error creating the operator roles: %s", err)
			os.Exit(1)
		}
	case "manual":
		reporter.Infof("Run the following commands to create the operator roles:\n")

		commands, err := buildCommands(reporter, prefix, cluster, creator.AccountID)
		if err != nil {
			reporter.Errorf("There was an error building the list of resources: %s", err)
			os.Exit(1)
		}
		fmt.Println(commands)
	}
}

func createRoles(reporter *rprtr.Object, awsClient aws.Client,
	prefix string, cluster *cmv1.Cluster, accountID string) error {
	version := getVersionMinor(cluster)

	for _, operator := range aws.CredentialRequests {
		roleName := getRoleName(cluster, operator)
		if roleName == "" {
			return fmt.Errorf("Failed to find operator IAM role")
		}

		if !confirm.Confirm("create the '%s' role", roleName) {
			continue
		}

		policy, err := generateRolePolicyDoc(cluster, accountID, operator)
		if err != nil {
			return err
		}

		reporter.Debugf("Creating role '%s'", roleName)
		roleARN, err := awsClient.EnsureRole(roleName, policy, map[string]string{
			tags.ClusterID:        cluster.ID(),
			tags.OpenShiftVersion: version,
			tags.RolePrefix:       prefix,
			"operator_namespace":  operator.Namespace,
			"operator_name":       operator.Name,
		})
		if err != nil {
			return err
		}
		reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)

		policyARN := getPolicyARN(accountID, prefix, operator.Namespace, operator.Name)

		reporter.Debugf("Attaching permission policy '%s' to role '%s'", policyARN, roleName)
		err = awsClient.AttachRolePolicy(roleName, policyARN)
		if err != nil {
			return err
		}
	}

	return nil
}

func buildCommands(reporter *rprtr.Object, prefix string, cluster *cmv1.Cluster, accountID string) (string, error) {
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
		createRole := fmt.Sprintf("aws iam create-role \\\n"+
			"\t--role-name %s \\\n"+
			"\t--assume-role-policy-document file://%s \\\n"+
			"\t--tags %s",
			roleName, filename, iamTags)
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
