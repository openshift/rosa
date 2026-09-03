package cluster

import (
	"context"
	"fmt"
	"os"
	"time"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	hfpathbind "github.com/openshift/rosa/pkg/hyperfleet/pathbind"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

// hfClusterUpdateInput is the backing store for hyperfleet-specific edit cluster flags.
var hfClusterUpdateInput hfpathbind.ClusterUpdateInput

var (
	hfEnabled     = hyperfleet.Enabled
	exitFn        = func(code int) { os.Exit(code) }
	hfEditCluster = func(cmd *cobra.Command) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleetEdit(r, cmd)
	}
)

// runHyperfleetEdit is a thin wrapper for direct test invocation.
func runHyperfleetEdit(r *rosa.Runtime, cmd *cobra.Command) {
	clusterKey, err := ocm.GetClusterKey()
	if err != nil || clusterKey == "" {
		r.Reporter.Errorf("--cluster is required")
		exitFn(1)
		return
	}

	clusterUID, err := hyperfleet.ResolveClusterUID(cmd.Context(), r.HyperFleetClient, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
		return
	}

	if err := hfpathbind.RunUpdateCluster(cmd.Context(), r, cmd, clusterUID, &hfClusterUpdateInput,
		&hyperfleetClusterUpdate{
			clusterKey: clusterKey,
			clusterUID: clusterUID,
			cmd:        cmd,
		},
	); err != nil {
		r.Reporter.Errorf("Failed to update cluster: %v", err)
		exitFn(1)
	}
}

// hyperfleetClusterUpdate implements hfpathbind.ClusterUpdateHandler for rosa edit cluster.
type hyperfleetClusterUpdate struct {
	hfpathbind.GeneratedClusterUpdatePrompt
	clusterKey string
	clusterUID string
	cmd        *cobra.Command
}

func (h *hyperfleetClusterUpdate) PreRequest(_ context.Context, _ *rosa.Runtime,
	input *hfpathbind.ClusterUpdateInput) error {
	if !h.cmd.Flags().Changed("expiration") && !h.cmd.Flags().Changed("expiration-time") &&
		!h.cmd.Flags().Changed("display-name") && !h.cmd.Flags().Changed("delete-protection") {
		return fmt.Errorf(
			"specify at least one supported flag: --expiration, --expiration-time, --display-name, --delete-protection",
		)
	}

	// --expiration-time and --expiration are OCM-registered (hidden) flags that
	// registerIfNew skips. Bridge via validateExpiration() which reads args.*.
	// --display-name and --delete-protection are new HF-only flags registered by
	// RegisterClusterUpdateFlags and arrive directly from cobra — no bridge needed.
	if h.cmd.Flags().Changed("expiration") || h.cmd.Flags().Changed("expiration-time") {
		expiry, err := validateExpiration()
		if err != nil {
			return err
		}
		if !expiry.IsZero() {
			input.ExpirationTimestamp = expiry.UTC().Format(time.RFC3339)
		}
	}
	return nil
}

func (h *hyperfleetClusterUpdate) PostExpand(
	ctx context.Context,
	r *rosa.Runtime,
	_ *hfpathbind.ClusterUpdateInput,
	obj *v1alpha1.Cluster,
) error {
	// Get current cluster to preserve fields not covered by this update.
	// The Platform API PUT expects the full spec; we merge only the changed
	// fields onto the fetched state so unreferenced fields are not zeroed.
	// TODO: this Get-then-merge could be eliminated if the Platform API
	// supports PATCH or the SDK bridge wrapper handles partial updates.
	current, err := r.HyperFleetClient.HyperfleetV1alpha1().Clusters().Get(ctx, h.clusterUID, platform.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get cluster %q: %w", h.clusterKey, err)
	}

	// Start from the current object — preserves Name, UID, and all existing spec fields.
	// The bridge wrapper routes the Update by obj.UID, which is carried over here.
	merged := current.DeepCopy()

	if h.cmd.Flags().Changed("expiration-time") || h.cmd.Flags().Changed("expiration") {
		merged.Spec.ExpirationTimestamp = obj.Spec.ExpirationTimestamp
	}
	if h.cmd.Flags().Changed("display-name") {
		merged.Spec.DisplayName = obj.Spec.DisplayName
	}
	if h.cmd.Flags().Changed("delete-protection") {
		merged.Spec.DeleteProtection = obj.Spec.DeleteProtection
	}

	*obj = *merged
	return nil
}

func (h *hyperfleetClusterUpdate) PostResponse(_ context.Context, r *rosa.Runtime, _ *v1alpha1.Cluster) error {
	r.Reporter.Infof("Updated cluster '%s'", h.clusterKey)
	return nil
}
