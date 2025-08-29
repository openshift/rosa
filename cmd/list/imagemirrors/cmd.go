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

package imagemirrors

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "image-mirrors"
	short   = "List cluster image mirrors"
	long    = "List image mirror configurations for a Hosted Control Plane cluster."
	example = `  # List all image mirrors on a cluster named "mycluster"
  rosa list image-mirrors --cluster=mycluster`
)

var (
	aliases = []string{"image-mirror"}
)

func NewListImageMirrorsCommand() *cobra.Command {
	options := NewListImageMirrorsOptions()
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Aliases: aliases,
		Example: example,
		Args:    cobra.NoArgs,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), ListImageMirrorsRunner(options)),
	}

	output.AddFlag(cmd)
	ocm.AddClusterFlag(cmd)
	arguments.AddProfileFlag(cmd.Flags())
	arguments.AddRegionFlag(cmd.Flags())
	return cmd
}

func ListImageMirrorsRunner(options *ListImageMirrorsOptions) rosa.CommandRunner {
	return func(_ context.Context, runtime *rosa.Runtime, cmd *cobra.Command, _ []string) error {
		clusterKey := runtime.GetClusterKey()

		cluster, err := runtime.OCMClient.GetCluster(clusterKey, runtime.Creator)
		if err != nil {
			return err
		}
		imageMirrors, err := runtime.OCMClient.ListImageMirrors(cluster.ID())
		if err != nil {
			return fmt.Errorf("Failed to list image mirrors: %v", err)
		}

		if output.HasFlag() {
			return output.Print(imageMirrors)
		}

		if len(imageMirrors) == 0 {
			runtime.Reporter.Infof("No image mirrors found for cluster '%s'", clusterKey)
			return nil
		}

		// Create tabwriter for formatted output
		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(writer, "ID\tTYPE\tSOURCE\tMIRRORS\n")

		for _, mirror := range imageMirrors {
			mirrors := ""
			if len(mirror.Mirrors()) > 0 {
				for i, m := range mirror.Mirrors() {
					if i > 0 {
						mirrors += ", "
					}
					mirrors += m
				}
			}

			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n",
				mirror.ID(),
				mirror.Type(),
				mirror.Source(),
				mirrors,
			)
		}

		writer.Flush()
		return nil
	}
}
