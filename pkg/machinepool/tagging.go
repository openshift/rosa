// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra" //nolint:depguard

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive" //nolint:depguard
	"github.com/openshift/rosa/pkg/rosa"        //nolint:depguard
	"github.com/openshift/rosa/pkg/tagging"
)

// prepareNodePoolTagEdit diffs the desired pool overrides against the customer-only tag view.
func prepareNodePoolTagEdit(cmd *cobra.Command, cluster *cmv1.Cluster, nodePool *cmv1.NodePool,
	r *rosa.Runtime) (map[string]*string, error) {
	requested := cmd.Flags().Changed("tags")
	if !requested && interactive.Enabled() && tagging.ValidateCluster(cluster) == nil {
		var err error
		requested, err = interactive.GetBool(interactive.Input{
			Question: "Update machine pool AWS tags", Default: false,
			Help: cmd.Flags().Lookup("tags").Usage,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to prompt for machine pool AWS tags: %w", err)
		}
	}
	if !requested {
		return nil, nil
	}
	if err := tagging.ValidateCluster(cluster); err != nil {
		return nil, err
	}

	var desired map[string]string
	if cmd.Flags().Changed("tags") {
		input, err := cmd.Flags().GetStringArray("tags")
		if err != nil {
			return nil, err
		}
		desired, err = tagging.ParseDesired(input)
		if err != nil {
			return nil, fmt.Errorf("invalid AWS tags: %w", err)
		}
	}
	clusterTags, err := r.OCMClient.GetClusterUserTags(cluster.ID())
	if err != nil {
		return nil, fmt.Errorf("failed to read cluster AWS tags: %w", err)
	}
	poolTags, err := r.OCMClient.GetNodePoolUserTags(cluster.ID(), nodePool.ID())
	if err != nil {
		return nil, fmt.Errorf("failed to read machine pool AWS tags: %w", err)
	}
	current := tagging.PoolOverrides(clusterTags, poolTags)
	if interactive.Enabled() {
		promptDefault := aws.FormatUserTagsList(current)
		if cmd.Flags().Changed("tags") {
			promptDefault = aws.FormatUserTagsList(desired)
		}
		input, err := interactive.GetString(interactive.Input{
			Question: "Machine pool AWS tags (complete pool-specific set)",
			Help:     cmd.Flags().Lookup("tags").Usage, Default: promptDefault,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to prompt for machine pool AWS tags: %w", err)
		}
		parts := aws.SplitUserTagList(input)
		if input == "" {
			parts = []string{""}
		}
		desired, err = tagging.ParseDesired(parts)
		if err != nil {
			return nil, fmt.Errorf("invalid AWS tags: %w", err)
		}
	}
	for key, value := range desired {
		if inherited, ok := clusterTags[key]; ok && inherited == value {
			delete(desired, key)
		}
	}
	return tagging.Diff(current, desired), nil
}

// onlyMachinePoolTagsChanged reports whether a node-pool update can contain only tag changes.
func onlyMachinePoolTagsChanged(cmd *cobra.Command) bool {
	allowed := 0
	for _, name := range []string{"tags", "cluster", "machinepool", "profile", "region", "yes", "interactive", "output"} {
		if cmd.Flags().Changed(name) {
			allowed++
		}
	}
	return allowed == cmd.Flags().NFlag()
}
