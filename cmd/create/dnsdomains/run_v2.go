package dnsdomains

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/rosa"
)

func runV2(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithHyperFleet()
	defer r.Cleanup()
	if err := createDNSDomainV2(cmd, r); err != nil {
		r.Reporter.Errorf("Failed to create dns domain: %v", err)
		os.Exit(1)
	}
}

func createDNSDomainV2(cmd *cobra.Command, r *rosa.Runtime) error {
	if cmd.Flags().Changed("hosted-cp") && !args.hostedCp {
		return fmt.Errorf("only HCP DNS domains are supported by Platform API v2")
	}
	domain, err := hyperfleet.NewDNSDomainClient(r.HyperFleetClient).Create(context.Background())
	if err != nil {
		return err
	}
	r.Reporter.Infof("DNS domain ‘%s’ has been created.", domain.ID)
	r.Reporter.Infof("To view all DNS domains, run 'rosa list dns-domains'")
	return nil
}
