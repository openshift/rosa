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

package oidcprovider

import (
	// nolint:gosec
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var modes []string = []string{"auto", "manual"}

var args struct {
	mode string
}

var Cmd = &cobra.Command{
	Use:     "oidc-provider",
	Aliases: []string{"oidcprovider"},
	Short:   "Create OIDC provider for an STS cluster.",
	Long:    "Create OIDC provider for operators to authenticate against in an STS cluster.",
	Example: `  # Create OIDC provider for cluster named "mycluster"
  rosa create oidc-provider --cluster=mycluster`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)

	flags.StringVar(
		&args.mode,
		"mode",
		modes[0],
		"How to perform the operation. Valid options are:\n"+
			"auto: OIDC provider will be created using the current AWS account\n"+
			"manual: Command to create the OIDC provider will be output",
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
	if len(argv) == 2 && !cmd.Flag("cluster").Changed {
		ocm.SetClusterKey(argv[0])
		args.mode = argv[1]

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

	if cluster.AWS().STS().RoleARN() == "" {
		reporter.Errorf("Cluster '%s' is not an STS cluster.", clusterKey)
		os.Exit(1)
	}

	mode := args.mode
	if interactive.Enabled() && !skipInteractive {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "OIDC provider creation mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  mode,
			Options:  modes,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid OIDC provider creation mode: %s", err)
			os.Exit(1)
		}
	}

	switch mode {
	case "auto":
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

		if len(missingRoles) > 0 {
			reporter.Errorf("Unable to find all required IAM roles for operators:\n%s\n\nSee "+
				"'rosa create operator-roles --help'",
				strings.Join(missingRoles, "\n"))
			os.Exit(1)
		}

		if cluster.State() != cmv1.ClusterStateWaiting && cluster.State() != cmv1.ClusterStatePending {
			reporter.Infof("Cluster '%s' is %s and does not need additional configuration.",
				clusterKey, cluster.State())
			os.Exit(0)
		}

		oidcEndpointURL := cluster.AWS().STS().OIDCEndpointURL()
		oidcProviderExists, err := awsClient.HasOpenIDConnectProvider(oidcEndpointURL, creator.AccountID)
		if err != nil {
			if strings.Contains(err.Error(), "AccessDenied") {
				reporter.Debugf("Failed to verify if OIDC provider exists: %s", err)
			} else {
				reporter.Errorf("Failed to verify if OIDC provider exists: %s", err)
				os.Exit(1)
			}
		}
		if oidcProviderExists {
			reporter.Warnf("Cluster '%s' already has OIDC provider but has not yet started installation. "+
				"Verify that the cluster operator roles exist and are configured correctly.", clusterKey)
			os.Exit(1)
		}

		ocmClient.LogEvent("ROSACreateOIDCProviderModeAuto")
		reporter.Infof("Creating OIDC provider using '%s'", creator.ARN)
		if !confirm.Prompt(true, "Create the OIDC provider for cluster '%s'?", clusterKey) {
			os.Exit(0)
		}
		err = createProvider(reporter, awsClient, cluster)
		if err != nil {
			reporter.Errorf("There was an error creating the OIDC provider: %s", err)
			os.Exit(1)
		}
	case "manual":
		ocmClient.LogEvent("ROSACreateOIDCProviderModeManual")

		commands, err := buildCommands(reporter, cluster)
		if err != nil {
			reporter.Errorf("There was an error building the list of resources: %s", err)
			os.Exit(1)
		}

		if reporter.IsTerminal() {
			reporter.Infof("Run the following commands to create the OIDC provider:\n")
		}

		fmt.Println(commands)
	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", modes)
		os.Exit(1)
	}
}

func createProvider(reporter *rprtr.Object, awsClient aws.Client, cluster *cmv1.Cluster) error {
	oidcEndpointURL := cluster.AWS().STS().OIDCEndpointURL()

	thumbprint, err := getThumbprint(oidcEndpointURL)
	if err != nil {
		return err
	}
	reporter.Debugf("Using thumbprint '%s'", thumbprint)

	oidcProviderARN, err := awsClient.CreateOpenIDConnectProvider(oidcEndpointURL, thumbprint)
	if err != nil {
		return err
	}
	reporter.Infof("Created OIDC provider with ARN '%s'", oidcProviderARN)

	return nil
}

func buildCommands(reporter *rprtr.Object, cluster *cmv1.Cluster) (string, error) {
	commands := []string{}

	oidcEndpointURL := cluster.AWS().STS().OIDCEndpointURL()

	thumbprint, err := getThumbprint(oidcEndpointURL)
	if err != nil {
		return "", err
	}
	reporter.Debugf("Using thumbprint '%s'", thumbprint)

	createOpenIDConnectProvider := fmt.Sprintf("aws iam create-open-id-connect-provider \\\n"+
		"\t--url %s \\\n"+
		"\t--client-id-list %s %s \\\n"+
		"\t--thumbprint-list %s",
		oidcEndpointURL, aws.OIDCClientIDOpenShift, aws.OIDCClientIDSTSAWS, thumbprint)
	commands = append(commands, createOpenIDConnectProvider)

	return strings.Join(commands, "\n\n"), nil
}

func getThumbprint(oidcEndpointURL string) (string, error) {
	connect, err := url.ParseRequestURI(oidcEndpointURL)
	if err != nil {
		return "", err
	}

	response, err := http.Get(fmt.Sprintf("https://%s:443", connect.Host))
	if err != nil {
		return "", err
	}

	certChain := response.TLS.PeerCertificates

	// Grab the CA in the chain
	for _, cert := range certChain {
		if cert.IsCA {
			return sha1Hash(cert.Raw), nil
		}
	}

	// Fall back to using the last certficiate in the chain
	cert := certChain[len(certChain)-1]
	return sha1Hash(cert.Raw), nil
}

// sha1Hash computes the SHA1 of the byte array and returns the hex encoding as a string.
func sha1Hash(data []byte) string {
	// nolint:gosec
	hasher := sha1.New()
	hasher.Write(data)
	hashed := hasher.Sum(nil)
	return hex.EncodeToString(hashed)
}

func validateOperatorRoles(awsClient aws.Client, cluster *cmv1.Cluster) ([]string, error) {
	var missingRoles []string

	operatorIAMRoles := cluster.AWS().STS().OperatorIAMRoles()

	if len(operatorIAMRoles) == 0 {
		return missingRoles, fmt.Errorf("No Operator IAM roles found for cluster '%s'", cluster.Name())
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
