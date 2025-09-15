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
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "image-mirror"
	short   = "Edit image mirror for a cluster"
	long    = "Edit an existing image mirror configuration for a Hosted Control Plane cluster by ID. Only the mirrors list can be updated."
	example = `  # Update mirrors for image mirror with ID "abc123" on cluster "mycluster"
  rosa edit image-mirror --cluster=mycluster abc123 \
    --mirrors=mirror.corp.com/team,backup.corp.com/team,new-mirror.corp.com/team

  # Alternative: using the --id flag
  rosa edit image-mirror --cluster=mycluster --id=abc123 \
    --mirrors=mirror.corp.com/team,backup.corp.com/team,new-mirror.corp.com/team`
)

var (
	aliases = []string{"image-mirrors"}
)

func NewEditImageMirrorCommand() *cobra.Command {
	options := NewEditImageMirrorOptions()
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Aliases: aliases,
		Example: example,
		Args:    cobra.MaximumNArgs(1),
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), EditImageMirrorRunner(options)),
	}

	flags := cmd.Flags()

	flags.StringVar(
		&options.Args().Id,
		"id",
		"",
		"ID of the image mirror configuration to edit",
	)

	flags.StringVar(
		&options.Args().Type,
		"type",
		"digest",
		"Type of image mirror (default: digest)",
	)

	flags.StringSliceVar(
		&options.Args().Mirrors,
		"mirrors",
		[]string{},
		"New list of mirror registries (comma-separated, required). This will replace the existing mirrors.",
	)

	_ = cmd.MarkFlagRequired("mirrors")

	ocm.AddClusterFlag(cmd)
	arguments.AddProfileFlag(cmd.Flags())
	arguments.AddRegionFlag(cmd.Flags())
	return cmd
}

func EditImageMirrorRunner(options *EditImageMirrorOptions) rosa.CommandRunner {
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
			return fmt.Errorf("Cluster '%s' is not ready. Image mirrors can only be edited on ready clusters", clusterKey)
		}

		if !cluster.Hypershift().Enabled() {
			return fmt.Errorf("Image mirrors are only supported on Hosted Control Plane clusters")
		}

		if len(args.Mirrors) == 0 {
			return fmt.Errorf("At least one mirror registry must be specified")
		}

		updatedMirror, err := runtime.OCMClient.UpdateImageMirror(
			cluster.ID(), args.Id, args.Mirrors, &args.Type)
		if err != nil {
			return fmt.Errorf("Failed to edit image mirror: %v", err)
		}
		runtime.Reporter.Infof("Image mirror '%s' has been updated on cluster '%s'",
			updatedMirror.ID(), clusterKey)
		runtime.Reporter.Infof("Source: %s", updatedMirror.Source())
		runtime.Reporter.Infof("Updated mirrors: %v", updatedMirror.Mirrors())

		return nil
	}
}
