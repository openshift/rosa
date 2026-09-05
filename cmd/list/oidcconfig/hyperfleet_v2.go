package oidcconfig

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

// hfEnabled, hfExitFn, and hfListOidcConfigs are package-level
// vars so tests can stub the hyperfleet dispatch path.
var (
	hfEnabled         = hyperfleet.Enabled
	hfExitFn          = func(code int) { os.Exit(code) }
	hfListOidcConfigs = func() {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleetList(r)
	}
)

// runHyperfleetList lists OIDC Configs via the Platform API v2.
func runHyperfleetList(r *rosa.Runtime) {
	ctx := context.Background()

	// Debug: Log the API call details
	r.Reporter.Debugf("Listing OIDC configs via Platform API")
	r.Reporter.Debugf("  Resource Path: /api/v0/oidc_configs")

	// List all OIDC configs for the current account
	oidcConfigList, err := r.HyperFleetClient.HyperfleetV1alpha1().OidcConfigs().List(
		ctx,
		platform.ListOptions{},
	)
	if err != nil {
		r.Reporter.Debugf("Platform API error: %v", err)
		r.Reporter.Debugf("This likely means the /oidc_configs route is not implemented in the Platform API backend")
		r.Reporter.Errorf("Failed to list OIDC configs: %v", err)
		hfExitFn(1)
		return
	}

	if output.HasFlag() {
		// Convert to a format suitable for JSON/YAML output
		configs := make([]map[string]interface{}, 0, len(oidcConfigList.Items))
		for _, config := range oidcConfigList.Items {
			configs = append(configs, map[string]interface{}{
				"id":              config.Name,
				"type":            config.Spec.Type,
				"issuer_url":      getIssuerUrl(&config),
				"secret_arn":      config.Spec.SecretArn,
				"installer_role":  config.Spec.InstallerRoleArn,
			})
		}
		err = output.Print(configs)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			hfExitFn(1)
		}
		return
	}

	if len(oidcConfigList.Items) == 0 {
		r.Reporter.Infof("There are no OIDC Configurations for your organization")
		return
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprintf(writer, "ID\tTYPE\tISSUER URL\tSECRET ARN\n")
	for _, config := range oidcConfigList.Items {
		managed := "managed"
		if config.Spec.Type == "unmanaged" {
			managed = "unmanaged"
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n",
			config.Name,
			managed,
			getIssuerUrl(&config),
			config.Spec.SecretArn,
		)
	}
	writer.Flush()
}

// getIssuerUrl returns the issuer URL from the spec
func getIssuerUrl(config *v1alpha1.OidcConfig) string {
	return config.Spec.IssuerUrl
}
