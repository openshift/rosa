package dnsdomains

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/rosa"
)

func runV2(_ *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithHyperFleet()
	defer r.Cleanup()
	if err := deleteDNSDomainV2(r, argv[0]); err != nil {
		r.Reporter.Errorf("Failed to delete dns domain '%s': %v", argv[0], err)
		os.Exit(1)
	}
}

func deleteDNSDomainV2(r *rosa.Runtime, id string) error {
	if err := hyperfleet.NewDNSDomainClient(r.HyperFleetClient).Delete(context.Background(), id); err != nil {
		return err
	}
	r.Reporter.Infof("Successfully deleted dns domain '%s'", id)
	return nil
}
