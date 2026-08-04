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
		os.Exit(1)
	}

	if !cmd.Flags().Changed("expiration") && !cmd.Flags().Changed("expiration-time") {
		r.Reporter.Errorf("specify at least one supported flag: --expiration, --expiration-time")
		os.Exit(1)
	}

	expiration, err := validateExpiration()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	clusters := r.HyperFleetClient.HyperfleetV1alpha1().Clusters(r.Creator.AccountID)

	list, err := clusters.List(ctx, wrappers.ListOptions{})
	if err != nil {
		r.Reporter.Errorf("Failed to list clusters: %v", err)
		os.Exit(1)
	}

	var found = -1
	for i := range list.Items {
		if list.Items[i].Name == clusterKey {
			found = i
			break
		}
	}
	if found == -1 {
		r.Reporter.Errorf("Cluster '%s' not found", clusterKey)
		os.Exit(1)
	}

	updated := list.Items[found].DeepCopy()
	if !expiration.IsZero() {
		t := metav1.NewTime(expiration)
		updated.Spec.ExpirationTimestamp = &t
	}

	if _, err = clusters.Update(ctx, updated, wrappers.UpdateOptions{}); err != nil {
		r.Reporter.Errorf("Failed to update cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	r.Reporter.Infof("Updated cluster '%s'", clusterKey)
}
