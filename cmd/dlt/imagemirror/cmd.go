/*
Copyright (c) 2025 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package imagemirror

import (
	"context"
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "image-mirror"
	short   = "Delete image mirror from a cluster"
	long    = "Delete an image mirror configuration from a Hosted Control Plane cluster by ID."
	example = `  # Delete image mirror with ID "abc123" from cluster "mycluster"
  rosa delete image-mirror --cluster=mycluster abc123

  # Delete without confirmation prompt
  rosa delete image-mirror --cluster=mycluster abc123 --yes

  # Alternative: using the --id flag
  rosa delete image-mirror --cluster=mycluster --id=abc123`
)

var (
	aliases = []string{"image-mirrors"}
)

func NewDeleteImageMirrorCommand() *cobra.Command {
	options := NewDeleteImageMirrorOptions()
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Aliases: aliases,
		Example: example,
		Args:    cobra.MaximumNArgs(1),
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), DeleteImageMirrorRunner(options)),
	}

	flags := cmd.Flags()

	flags.StringVar(
		&options.Args().Id,
		"id",
		"",
		"ID of the image mirror configuration to delete",
	)

	flags.BoolVarP(
		&options.Args().Yes,
		"yes",
		"y",
		false,
		"Automatically answer yes to confirm deletion",
	)

	ocm.AddClusterFlag(cmd)
	arguments.AddProfileFlag(cmd.Flags())
	arguments.AddRegionFlag(cmd.Flags())
	return cmd
}

func DeleteImageMirrorRunner(options *DeleteImageMirrorOptions) rosa.CommandRunner {
	return func(_ context.Context, runtime *rosa.Runtime, cmd *cobra.Command, argv []string) error {
		clusterKey := runtime.GetClusterKey()
		args := options.Args()

		// Get image mirror ID from positional argument or flag
		if len(argv) == 1 && !cmd.Flag("id").Changed {
			args.Id = argv[0]
		}

		if args.Id == "" {
			return fmt.Errorf("Image mirror ID is required. Specify it as an argument or use the --id flag")
		}

		cluster, err := runtime.OCMClient.GetCluster(clusterKey, runtime.Creator)
		if err != nil {
			return err
		}
		if cluster.State() != cmv1.ClusterStateReady {
			return fmt.Errorf("Cluster '%s' is not ready. Image mirrors can only be deleted on ready clusters", clusterKey)
		}

		if !cluster.Hypershift().Enabled() {
			return fmt.Errorf("Image mirrors are only supported on Hosted Control Plane clusters")
		}

		// First, verify the image mirror exists
		imageMirror, err := runtime.OCMClient.GetImageMirror(cluster.ID(), args.Id)
		if err != nil {
			return fmt.Errorf("Failed to get image mirror '%s': %v", args.Id, err)
		}

		if !args.Yes {
			prompt := fmt.Sprintf("Are you sure you want to delete image mirror '%s' on cluster '%s'?",
				args.Id, clusterKey)
			confirmed, err := interactive.GetBool(interactive.Input{
				Question: prompt,
				Default:  false,
				Required: false,
			})
			if err != nil {
				return err
			}
			if !confirmed {
				return nil
			}
		}

		err = runtime.OCMClient.DeleteImageMirror(cluster.ID(), args.Id)
		if err != nil {
			return fmt.Errorf("Failed to delete image mirror: %v", err)
		}

		runtime.Reporter.Infof("Image mirror '%s' has been deleted from cluster '%s'",
			imageMirror.ID(), clusterKey)

		return nil
	}
}
