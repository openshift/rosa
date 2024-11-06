/*
Copyright (c) 2023 Red Hat, Inc.

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

package dnsdomains

import (
	// nolint:gosec
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	hostedCp bool
}

var Cmd = &cobra.Command{
	Use:     "dns-domain",
	Aliases: []string{"dnsdomain"},
	Short:   "Create DNS Domain.",
	Long:    "Create DNS Domain.",
	Example: `  # Create DNS Domain
	rosa create dns-domain`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()

	flags.BoolVar(
		&args.hostedCp,
		"hosted-cp",
		false,
		"If creating a dns-domain for a Hosted Control Plane cluster",
	)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	domain, err := createDnsDomain(args.hostedCp)
	if err != nil {
		r.Reporter.Errorf("Failed to build DNS domain: %s", err)
		os.Exit(1)
	}
	dnsdomain, err := r.OCMClient.CreateDNSDomain(domain)
	if err != nil {
		r.Reporter.Errorf("Failed to create dns domain: %s", err)
		os.Exit(1)
	}

	r.Reporter.Infof("DNS domain ‘%s’ has been created.", dnsdomain.ID())
	r.Reporter.Infof("To view all DNS domains, run 'rosa list dns-domains")
}

func createDnsDomain(isHostedCp bool) (*cmv1.DNSDomain, error) {
	dnsDomainBuilder := cmv1.DNSDomainBuilder{}
	if isHostedCp {
		dnsDomainBuilder.ClusterArch(cmv1.ClusterArchitectureHcp)
	} else {
		dnsDomainBuilder.ClusterArch(cmv1.ClusterArchitectureClassic)
	}

	return dnsDomainBuilder.Build()
}
