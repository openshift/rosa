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
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	all      bool
	hostedCp bool
}

var Cmd = &cobra.Command{
	Use:     "dns-domain",
	Aliases: []string{"dnsdomain", "dnsdomains", "dns-domain", "dns-domains"},
	Short:   "List DNS Domains",
	Long:    "List DNS Domains",
	Example: `  # List all DNS Domains tied to your organization ID"
  rosa list dns-domain`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()

	flags.BoolVarP(
		&args.all,
		"all",
		"a",
		false,
		"List all DNS domains (default lists just user defined).",
	)

	flags.BoolVar(
		&args.hostedCp,
		"hosted-cp",
		false,
		"Filter to list only DNS Domains used for Hosted Control Plane clusters",
	)

	output.AddFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	r.Reporter.Debugf("Loading dns domains for current org id")
	search := "user_defined='true'"
	if args.all {
		search = ""
	}
	dnsDomains, err := r.OCMClient.ListDNSDomains(search)
	if err != nil {
		r.Reporter.Errorf("Failed to list DNS Domains: %v", err)
		os.Exit(1)
	}

	if args.hostedCp {
		dnsDomains = filterByClusterArch(dnsDomains, v1.ClusterArchitectureHcp)
	}

	if output.HasFlag() {
		err = output.Print(dnsDomains)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(dnsDomains) == 0 {
		r.Reporter.Infof("There are no DNS Domains for your organization")
		os.Exit(0)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprintf(writer, "ID\tCLUSTER ID\tRESERVED TIME\tUSER DEFINED\tARCHITECTURE\n")
	for _, dnsdomain := range dnsDomains {
		userDefined := "No"
		if dnsdomain.UserDefined() {
			userDefined = "Yes"
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n",
			dnsdomain.ID(),
			dnsdomain.Cluster().ID(),
			dnsdomain.ReservedAtTimestamp().Format(time.RFC3339),
			userDefined,
			dnsdomain.ClusterArch(),
		)
	}
	writer.Flush()
}

func filterByClusterArch(domains []*v1.DNSDomain, arch v1.ClusterArchitecture) []*v1.DNSDomain {
	finalDomains := make([]*v1.DNSDomain, 0)
	for _, domain := range domains {
		if domain.ClusterArch() == arch {
			finalDomains = append(finalDomains, domain)
		}
	}
	return finalDomains
}
