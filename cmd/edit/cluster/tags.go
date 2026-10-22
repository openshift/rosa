// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/tagging"
)

// prepareClusterTagEdit validates the cluster and computes nullable changes from the desired customer tags.
func prepareClusterTagEdit(r *rosa.Runtime, cmd *cobra.Command, cluster *cmv1.Cluster) (map[string]*string, error) {
	var help string
	if flag := cmd.Flags().Lookup("tags"); flag != nil {
		help = flag.Usage
	}
	return tagging.PrepareEdit(tagging.EditOptions{
		Cluster:             cluster,
		TagsChanged:         cmd.Flags().Changed("tags"),
		Input:               args.tags,
		OptInQuestion:       "Update customer AWS tags",
		DesiredQuestion:     "Customer AWS tags (complete desired set)",
		Help:                help,
		OptInErrorContext:   "AWS tag editing",
		DesiredErrorContext: "AWS tags",
		ReadCurrent: func() (map[string]string, map[string]string, error) {
			current, err := r.OCMClient.GetClusterUserTags(cluster.ID())
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read cluster AWS tags: %w", err)
			}
			return current, nil, nil
		},
	})
}

// onlyClusterTagsChanged reports whether the command requests no edit besides AWS tags.
func onlyClusterTagsChanged(cmd *cobra.Command) bool {
	return tagging.OnlyTagsChanged(cmd.Flags().NFlag(), cmd.Flags().Changed,
		"cluster", "profile", "region", "yes", "interactive")
}
