package cluster

import (
	"context"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-online/rosa-hyperfleet-api/clientset/wrappers"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hfEnabled     = hyperfleet.Enabled
	exitFn        = func(code int) { os.Exit(code) }
	hfEditCluster = func(cmd *cobra.Command) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleetEdit(r, cmd)
	}
)

// runHyperfleetEdit updates a hyperfleet cluster's spec via the Platform API.
// Only --expiration and --expiration-time are supported; all other edit flags
// are OCM-only and have no equivalent in the Platform API spec.
func runHyperfleetEdit(r *rosa.Runtime, cmd *cobra.Command) {
	ctx := context.Background()

	clusterKey, err := ocm.GetClusterKey()
	if err != nil || clusterKey == "" {
		r.Reporter.Errorf("--cluster is required")
		exitFn(1)
	}

	if !cmd.Flags().Changed("expiration") && !cmd.Flags().Changed("expiration-time") {
		r.Reporter.Errorf("specify at least one supported flag: --expiration, --expiration-time")
		exitFn(1)
	}

	expiration, err := validateExpiration()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		exitFn(1)
	}

	clusterID, err := hyperfleet.ResolveClusterUID(ctx, r.HyperFleetClient, r.Creator.AccountID, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
	}

	clusters := r.HyperFleetClient.HyperfleetV1alpha1().Clusters(r.Creator.AccountID)
	current, err := clusters.Get(ctx, clusterID, wrappers.GetOptions{})
	if err != nil {
		r.Reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		exitFn(1)
	}

	updated := current.DeepCopy()
	if !expiration.IsZero() {
		t := metav1.NewTime(expiration)
		updated.Spec.ExpirationTimestamp = &t
	}

	if _, err = clusters.Update(ctx, updated, wrappers.UpdateOptions{}); err != nil {
		r.Reporter.Errorf("Failed to update cluster '%s': %v", clusterKey, err)
		exitFn(1)
	}

	r.Reporter.Infof("Updated cluster '%s'", clusterKey)
}
