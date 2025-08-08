package rhRegion

import (
	"fmt"
	"os"
	"strings"

	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	args struct {
		discoveryURL string
	}
)

var Cmd = NewRhRegionCommand()

func NewRhRegionCommand() *cobra.Command {
	Cmd := &cobra.Command{
		Use:   "rh-regions",
		Short: "List available OCM regions",
		Long:  "The command lists available OpenShift Cluster Manager regions.",
		Example: `  # List all supported OCM regions 
ocm list rh-regions`,
		Run:    run,
		Hidden: true,
		Args:   cobra.NoArgs,
	}

	flags := Cmd.Flags()
	flags.StringVar(
		&args.discoveryURL,
		"discovery-url",
		"",
		"URL of the OCM API gateway. If not provided, will reuse the URL from the configuration "+
			"file or "+sdk.DefaultURL+" as a last resort. The value should be a complete URL "+
			"or a valid URL alias: "+strings.Join(ocm.ValidOCMUrlAliases(), ", "),
	)
	return Cmd
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime()

	err := ListRhRegions(args.discoveryURL, r)
	if err != nil {
		r.Reporter.Errorf("Failed to determine gateway URL: %v", err)
		os.Exit(1)
	}
}

func ListRhRegions(discoveryURL string, r *rosa.Runtime) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("Failed to load config file: %v", err)
	}

	gatewayURL, err := ocm.ResolveGatewayUrl(discoveryURL, cfg)
	if err != nil {
		return fmt.Errorf("Failed to determine gateway URL: %v", err)
	}

	// Create the writer that will be used to print the tabulated results:
	tb := output.NewTableBuilder()

	fmt.Printf("Discovery URL: %s\n\n", gatewayURL)
	regions, err := sdk.GetRhRegions(gatewayURL)
	if err != nil {
		return fmt.Errorf("Failed to get OCM regions: %v", err)
	}

	// If there are no regions, print a warning message and return early
	if len(regions) == 0 {
		r.Reporter.Warnf("No regions found")
		return nil
	}

	tb.SetHeaders("RH Region", "Gateway URL")
	for regionName, region := range regions {
		tb.AddRow(regionName, region.URL)
	}
	tb.Render()
	return nil
}
