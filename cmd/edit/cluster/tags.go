// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/tagging"
)

// prepareClusterTagEdit validates the cluster and computes nullable changes from the desired customer tags.
func prepareClusterTagEdit(r *rosa.Runtime, cmd *cobra.Command, cluster *cmv1.Cluster) (map[string]*string, error) {
	requested := cmd.Flags().Changed("tags")
	if !requested && interactive.Enabled() && tagging.ValidateCluster(cluster) == nil {
		var err error
		requested, err = interactive.GetBool(interactive.Input{
			Question: "Update customer AWS tags", Default: false,
			Help: cmd.Flags().Lookup("tags").Usage,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to prompt for AWS tag editing: %w", err)
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
		var err error
		desired, err = tagging.ParseDesired(args.tags)
		if err != nil {
			return nil, fmt.Errorf("invalid AWS tags: %w", err)
		}
	}
	current, err := r.OCMClient.GetClusterUserTags(cluster.ID())
	if err != nil {
		return nil, fmt.Errorf("failed to read cluster AWS tags: %w", err)
	}
	if interactive.Enabled() {
		promptDefault := aws.FormatUserTagsList(current)
		if cmd.Flags().Changed("tags") {
			promptDefault = aws.FormatUserTagsList(desired)
		}
		input, err := interactive.GetString(interactive.Input{
			Question: "Customer AWS tags (complete desired set)",
			Help:     cmd.Flags().Lookup("tags").Usage, Default: promptDefault,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to prompt for AWS tags: %w", err)
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
	return tagging.Diff(current, desired), nil
}

// onlyClusterTagsChanged reports whether the command requests no edit besides AWS tags.
func onlyClusterTagsChanged(cmd *cobra.Command) bool {
	onlyTags := true
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		switch flag.Name {
		case "tags", "cluster", "profile", "region", "yes", "interactive":
		default:
			onlyTags = false
		}
	})
	return onlyTags
}
