package dnsdomains

import (
	"context"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

func runV2(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithHyperFleet()
	defer r.Cleanup()
	if err := listDNSDomainsV2(r, os.Stdout); err != nil {
		r.Reporter.Errorf("Failed to list DNS domains: %v", err)
		os.Exit(1)
	}
}

func listDNSDomainsV2(r *rosa.Runtime, out io.Writer) error {
	// All domains exposed by the Platform API are account-scoped and HCP.
	// --all and --hosted-cp therefore select the same collection.
	domains, err := hyperfleet.NewDNSDomainClient(r.HyperFleetClient).List(context.Background())
	if err != nil {
		return err
	}
	if output.HasFlag() {
		return output.Print(domains)
	}
	if len(domains) == 0 {
		r.Reporter.Infof("There are no DNS Domains for your account")
		return nil
	}
	return printDNSDomainsV2(out, domains)
}

func printDNSDomainsV2(out io.Writer, domains []hyperfleet.DNSDomain) error {
	writer := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tCLUSTER ID\tRESERVED TIME\tUSER DEFINED\tARCHITECTURE")
	for _, domain := range domains {
		userDefined := "No"
		if domain.UserDefined {
			userDefined = "Yes"
		}
		clusterID := ""
		if domain.Cluster != nil {
			clusterID = domain.Cluster.ID
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", domain.ID, clusterID,
			domain.ReservedAtTimestamp.Format(time.RFC3339), userDefined, domain.ClusterArch)
	}
	return writer.Flush()
}
