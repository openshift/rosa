package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/types"

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

// supportedChannelGroups are channel-group values accepted by V2 edit.
// "nightly" is recognized but rejected without a V2 version catalog (parity with OCM).
var supportedChannelGroups = map[string]struct{}{
	"stable":    {},
	"candidate": {},
	"fast":      {},
	"eus":       {},
	"nightly":   {},
}

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

	h := &hyperfleetClusterUpdate{
		clusterKey: clusterKey,
		clusterUID: clusterUID,
		cmd:        cmd,
	}
	if err := h.PreRequest(cmd.Context(), r, &hfClusterUpdateInput); err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
		return
	}

	// Platform API update is a merge of only the fields present in the request
	// body. Sending a full Get→PUT round-trip fails validation because the public
	// projection includes service-set fields (issuerURL, empty release.image, …).
	// HostedCluster also lacks omitempty, so a typed Update always serializes an
	// empty nested object. Patch with an explicit partial spec avoids both issues.
	patch, err := h.buildSpecPatch(&hfClusterUpdateInput)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
		return
	}

	updated, err := r.HyperFleetClient.HyperfleetV1alpha1().Clusters().Patch(
		cmd.Context(), clusterUID, types.MergePatchType, patch, platform.PatchOptions{})
	if err != nil {
		r.Reporter.Errorf("Failed to update cluster: %v", err)
		exitFn(1)
		return
	}
	if err := h.PostResponse(cmd.Context(), r, updated); err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
	}
}

// hyperfleetClusterUpdate implements pre/post hooks for rosa edit cluster (HF).
type hyperfleetClusterUpdate struct {
	clusterKey string
	clusterUID string
	cmd        *cobra.Command
}

func (h *hyperfleetClusterUpdate) PreRequest(_ context.Context, _ *rosa.Runtime,
	input *hfpathbind.ClusterUpdateInput) error {
	channelChanged := h.cmd.Flags().Changed("channel-group") || h.cmd.Flags().Changed("channel")
	if !h.cmd.Flags().Changed("expiration") && !h.cmd.Flags().Changed("expiration-time") &&
		!h.cmd.Flags().Changed("display-name") && !h.cmd.Flags().Changed("delete-protection") &&
		!channelChanged {
		return fmt.Errorf(
			"specify at least one supported flag: --expiration, --expiration-time, " +
				"--display-name, --delete-protection, --channel-group, --channel",
		)
	}

	// --expiration-time and --expiration are OCM-registered (hidden) flags that
	// registerIfNew skips. Bridge via validateExpiration() which reads args.*.
	// --display-name and --delete-protection are new HF-only flags registered by
	// RegisterClusterUpdateFlags and arrive directly from cobra — no bridge needed.
	// --channel-group / --channel are OCM-registered; bridged in buildSpecPatch via args.*.
	if h.cmd.Flags().Changed("expiration") || h.cmd.Flags().Changed("expiration-time") {
		expiry, err := validateExpiration()
		if err != nil {
			return err
		}
		if !expiry.IsZero() {
			input.ExpirationTimestamp = expiry.UTC().Format(time.RFC3339)
		}
	}

	if channelChanged {
		if err := validateHyperfleetChannelArgs(); err != nil {
			return err
		}
	}
	return nil
}

func validateHyperfleetChannelArgs() error {
	// Prefer --channel-group (OCM-style group name). --channel is the versioned
	// channel id (e.g. stable-4.20); accept any non-empty value for that flag.
	if args.channelGroup != "" {
		if _, ok := supportedChannelGroups[args.channelGroup]; !ok {
			// Capitalized to match OCM's user-facing error wording (e2e ContainSubstring).
			return fmt.Errorf("Unsupported channel group '%s'", args.channelGroup) //nolint:staticcheck // ST1005: OCM parity
		}
		// Without a V2 version catalog, nightly cannot be verified for the
		// cluster's current version — match OCM's "not available" wording.
		if args.channelGroup == "nightly" {
			return fmt.Errorf("is not available for the desired channel group")
		}
	}
	return nil
}

// buildSpecPatch returns a merge-patch body with only the fields the user changed.
func (h *hyperfleetClusterUpdate) buildSpecPatch(input *hfpathbind.ClusterUpdateInput) ([]byte, error) {
	spec := map[string]any{}

	if h.cmd.Flags().Changed("expiration-time") || h.cmd.Flags().Changed("expiration") {
		if input.ExpirationTimestamp != "" {
			spec["expirationTimestamp"] = input.ExpirationTimestamp
		}
	}
	if h.cmd.Flags().Changed("display-name") {
		spec["displayName"] = input.DisplayName
	}
	if h.cmd.Flags().Changed("delete-protection") {
		if input.DeleteProtection == nil {
			return nil, fmt.Errorf("delete-protection flag is set but value is missing")
		}
		spec["deleteProtection"] = *input.DeleteProtection
	}
	if h.cmd.Flags().Changed("channel-group") {
		spec["properties"] = map[string]any{"channel_group": args.channelGroup}
	}
	if h.cmd.Flags().Changed("channel") {
		spec["hostedCluster"] = map[string]any{"channel": args.channel}
	}

	if len(spec) == 0 {
		return nil, fmt.Errorf("no supported fields to update")
	}
	return json.Marshal(map[string]any{"spec": spec})
}

func (h *hyperfleetClusterUpdate) PostResponse(_ context.Context, r *rosa.Runtime, _ *v1alpha1.Cluster) error {
	r.Reporter.Infof("Updated cluster '%s'", h.clusterKey)
	return nil
}
