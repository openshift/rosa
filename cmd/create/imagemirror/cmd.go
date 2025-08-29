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
	short   = "Create image mirror for a cluster"
	long    = "Create an image mirror configuration for a Hosted Control Plane cluster. The image mirror ID will be auto-generated."
	example = `  # Create an image mirror for cluster "mycluster"
  rosa create image-mirror --cluster=mycluster \
    --source=registry.example.com/team \
    --mirrors=mirror.corp.com/team,backup.corp.com/team

  # Create with a specific type (digest is default and only supported type)
  rosa create image-mirror --cluster=mycluster \
    --type=digest --source=docker.io/library \
    --mirrors=internal-registry.company.com/dockerhub`
)

var (
	aliases = []string{"image-mirrors"}
)

func NewCreateImageMirrorCommand() *cobra.Command {
	options := NewCreateImageMirrorOptions()
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Aliases: aliases,
		Example: example,
		Args:    cobra.NoArgs,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), CreateImageMirrorRunner(options)),
	}

	flags := cmd.Flags()

	flags.StringVar(
		&options.Args().Type,
		"type",
		"digest",
		"Type of image mirror (default: digest)",
	)

	flags.StringVar(
		&options.Args().Source,
		"source",
		"",
		"Source registry that will be mirrored (required)",
	)

	flags.StringSliceVar(
		&options.Args().Mirrors,
		"mirrors",
		[]string{},
		"List of mirror registries (comma-separated, required)",
	)

	_ = cmd.MarkFlagRequired("source")
	_ = cmd.MarkFlagRequired("mirrors")

	ocm.AddClusterFlag(cmd)
	arguments.AddProfileFlag(cmd.Flags())
	arguments.AddRegionFlag(cmd.Flags())
	return cmd
}

func CreateImageMirrorRunner(options *CreateImageMirrorOptions) rosa.CommandRunner {
	return func(_ context.Context, runtime *rosa.Runtime, cmd *cobra.Command, _ []string) error {
		clusterKey := runtime.GetClusterKey()
		args := options.Args()

		cluster, err := runtime.OCMClient.GetCluster(clusterKey, runtime.Creator)
		if err != nil {
			return err
		}
		if cluster.State() != cmv1.ClusterStateReady {
			return fmt.Errorf("Cluster '%s' is not ready. Image mirrors can only be created on ready clusters", clusterKey)
		}

		if !cluster.Hypershift().Enabled() {
			return fmt.Errorf("Image mirrors are only supported on Hosted Control Plane clusters")
		}

		if len(args.Mirrors) == 0 {
			return fmt.Errorf("At least one mirror registry must be specified")
		}

		createdMirror, err := runtime.OCMClient.CreateImageMirror(
			cluster.ID(), args.Type, args.Source, args.Mirrors)
		if err != nil {
			return fmt.Errorf("Failed to create image mirror: %v", err)
		}
		runtime.Reporter.Infof("Image mirror with ID '%s' has been created on cluster '%s'",
			createdMirror.ID(), clusterKey)
		runtime.Reporter.Infof("Source: %s", createdMirror.Source())
		runtime.Reporter.Infof("Mirrors: %v", createdMirror.Mirrors())

		return nil
	}
}
