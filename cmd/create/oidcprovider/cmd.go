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
	"bytes"
	"crypto/sha1" //#nosec GSC-G505 -- Import blacklist: crypto/sha1
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

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
	aws.AddModeFlag(Cmd)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	// Allow the command to be called programmatically
	skipInteractive := false
	if len(argv) == 2 && !cmd.Flag("cluster").Changed {
		ocm.SetClusterKey(argv[0])
		aws.SetModeKey(argv[1])

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
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") {
		interactive.Enable()
	}

	cluster := r.FetchCluster()
	if cluster.AWS().STS().RoleARN() == "" {
		r.Reporter.Errorf("Cluster '%s' is not an STS cluster.", clusterKey)
		os.Exit(1)
	}

	if interactive.Enabled() && !skipInteractive {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "OIDC provider creation mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  aws.ModeAuto,
			Options:  aws.Modes,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid OIDC provider creation mode: %s", err)
			os.Exit(1)
		}
	}

	switch mode {
	case aws.ModeAuto:
		if cluster.State() != cmv1.ClusterStateWaiting && cluster.State() != cmv1.ClusterStatePending {
			r.Reporter.Infof("Cluster '%s' is %s and does not need additional configuration.",
				clusterKey, cluster.State())
			os.Exit(0)
		}
		oidcEndpointURL := cluster.AWS().STS().OIDCEndpointURL()
		oidcProviderExists, err := r.AWSClient.HasOpenIDConnectProvider(oidcEndpointURL, r.Creator.AccountID)
		if err != nil {
			if strings.Contains(err.Error(), "AccessDenied") {
				r.Reporter.Debugf("Failed to verify if OIDC provider exists: %s", err)
			} else {
				r.Reporter.Errorf("Failed to verify if OIDC provider exists: %s", err)
				os.Exit(1)
			}
		}
		if oidcProviderExists {
			r.Reporter.Warnf("Cluster '%s' already has OIDC provider but has not yet started installation. "+
				"Verify that the cluster operator roles exist and are configured correctly.", clusterKey)
			os.Exit(1)
		}
		if !output.HasFlag() || r.Reporter.IsTerminal() {
			r.Reporter.Infof("Creating OIDC provider using '%s'", r.Creator.ARN)
		}
		if !confirm.Prompt(true, "Create the OIDC provider for cluster '%s'?", clusterKey) {
			os.Exit(0)
		}
		err = createProvider(r, cluster)
		if err != nil {
			r.Reporter.Errorf("There was an error creating the OIDC provider: %s", err)
			r.OCMClient.LogEvent("ROSACreateOIDCProviderModeAuto", map[string]string{
				ocm.ClusterID: clusterKey,
				ocm.Response:  ocm.Failure,
			})
			os.Exit(1)
		}
		r.OCMClient.LogEvent("ROSACreateOIDCProviderModeAuto", map[string]string{
			ocm.ClusterID: clusterKey,
			ocm.Response:  ocm.Success,
		})
	case aws.ModeManual:

		commands, err := buildCommands(r, cluster)
		if err != nil {
			r.Reporter.Errorf("There was an error building the list of resources: %s", err)
			os.Exit(1)
			r.OCMClient.LogEvent("ROSACreateOIDCProviderModeManual", map[string]string{
				ocm.ClusterID: clusterKey,
				ocm.Response:  ocm.Failure,
			})
		}
		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("Run the following commands to create the OIDC provider:\n")
		}
		r.OCMClient.LogEvent("ROSACreateOIDCProviderModeManual", map[string]string{
			ocm.ClusterID: clusterKey,
		})
		fmt.Println(commands)
	default:
		r.Reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
}

func createProvider(r *rosa.Runtime, cluster *cmv1.Cluster) error {
	oidcEndpointURL := cluster.AWS().STS().OIDCEndpointURL()

	thumbprint, err := getThumbprint(oidcEndpointURL)
	if err != nil {
		return err
	}
	r.Reporter.Debugf("Using thumbprint '%s'", thumbprint)

	oidcProviderARN, err := r.AWSClient.CreateOpenIDConnectProvider(oidcEndpointURL, thumbprint, cluster.ID())
	if err != nil {
		return err
	}
	if !output.HasFlag() || r.Reporter.IsTerminal() {
		r.Reporter.Infof("Created OIDC provider with ARN '%s'", oidcProviderARN)
	}

	return nil
}

func buildCommands(r *rosa.Runtime, cluster *cmv1.Cluster) (string, error) {
	commands := []string{}

	oidcEndpointURL := cluster.AWS().STS().OIDCEndpointURL()

	thumbprint, err := getThumbprint(oidcEndpointURL)
	if err != nil {
		return "", err
	}
	r.Reporter.Debugf("Using thumbprint '%s'", thumbprint)

	tag := fmt.Sprintf(
		"Key=%s,Value=%s",
		tags.ClusterID, cluster.ID())
	createOpenIDConnectProvider := fmt.Sprintf("aws iam create-open-id-connect-provider \\\n"+
		"\t--url %s \\\n"+
		"\t--client-id-list %s %s \\\n"+
		"\t--thumbprint-list %s \\\n"+
		"\t --tag %s",
		oidcEndpointURL, aws.OIDCClientIDOpenShift, aws.OIDCClientIDSTSAWS, thumbprint, tag)
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
			if bytes.Equal(cert.RawIssuer, cert.RawSubject) {
				return sha1Hash(cert.Raw), nil
			}
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
