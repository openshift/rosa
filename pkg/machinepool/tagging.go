// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra" //nolint:depguard

	"github.com/openshift/rosa/pkg/interactive" //nolint:depguard
	"github.com/openshift/rosa/pkg/rosa"        //nolint:depguard
	"github.com/openshift/rosa/pkg/tagging"
)

// prepareNodePoolTagEdit diffs the desired pool overrides against the customer-only tag view.
func prepareNodePoolTagEdit(cmd *cobra.Command, cluster *cmv1.Cluster, nodePool *cmv1.NodePool,
	r *rosa.Runtime) (map[string]*string, error) {
	var input []string
	if cmd.Flags().Changed("tags") {
		var err error
		input, err = cmd.Flags().GetStringArray("tags")
		if err != nil {
			return nil, err
		}
	}
	var help string
	if flag := cmd.Flags().Lookup("tags"); flag != nil {
		help = flag.Usage
	}
	return tagging.PrepareEdit(tagging.EditOptions{
		Cluster:             cluster,
		TagsChanged:         cmd.Flags().Changed("tags"),
		Input:               input,
		OptInQuestion:       "Update machine pool AWS tags",
		DesiredQuestion:     "Machine pool AWS tags (complete pool-specific set)",
		Help:                help,
		OptInErrorContext:   "machine pool AWS tags",
		DesiredErrorContext: "machine pool AWS tags",
		OnInheritedMatch: func(key, value string) {
			r.Reporter.Warnf("Tag %q matches an existing cluster tag and won't be added as a machine pool "+
				"override; a later cluster tag change may change it.", key+":"+value)
		},
		ReadCurrent: func() (map[string]string, map[string]string, error) {
			clusterTags, err := r.OCMClient.GetClusterUserTags(cluster.ID())
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read cluster AWS tags: %w", err)
			}
			poolTags, err := r.OCMClient.GetNodePoolUserTags(cluster.ID(), nodePool.ID())
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read machine pool AWS tags: %w", err)
			}
			return poolTags, clusterTags, nil
		},
	})
}

// onlyMachinePoolTagsChanged reports whether a node-pool update can contain only tag changes.
func onlyMachinePoolTagsChanged(cmd *cobra.Command) bool {
	if interactive.Enabled() {
		return false
	}
	return tagging.OnlyTagsChanged(cmd.Flags().NFlag(), cmd.Flags().Changed,
		"cluster", "machinepool", "profile", "region", "yes", "interactive", "output")
}
