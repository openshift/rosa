/*
Copyright (c) 2020 Red Hat, Inc.

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

package logforwarders

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/fedramp"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "log-forwarders -c <cluster-id>"
	short   = "List cluster log forwarders"
	long    = "List log forwarders configured on a cluster, given a cluster ID"
	example = "  # List all log forwarders on a cluster named \"mycluster\": " +
		"rosa list log-forwarders --cluster=mycluster"
)

var aliases = []string{"logforwarders", "log-forwarder", "logforwarder"}

func NewListLogForwardersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Aliases: aliases,
		Example: example,
		Args:    cobra.NoArgs,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), ListLogForwardersRunner()),
	}

	ocm.AddClusterFlag(cmd)
	output.AddFlag(cmd)
	return cmd
}

func ListLogForwardersRunner() rosa.CommandRunner {
	return func(_ context.Context, runtime *rosa.Runtime, cmd *cobra.Command, _ []string) error {
		clusterKey := runtime.GetClusterKey()

		if fedramp.Enabled() {
			return fmt.Errorf("log forwarding is not supported on Govcloud")
		}

		cluster, err := runtime.OCMClient.GetCluster(clusterKey, runtime.Creator)
		if err != nil {
			return err
		}
		if cluster.State() != cmv1.ClusterStateReady &&
			cluster.State() != cmv1.ClusterStateHibernating {
			return fmt.Errorf("cluster '%s' is not yet ready", clusterKey)
		}

		runtime.Reporter.Debugf("Loading log forwarders for cluster '%s'", clusterKey)
		logForwarders, err := runtime.OCMClient.GetLogForwarders(cluster.ID())
		if err != nil {
			return fmt.Errorf("failed to get log forwarders for cluster '%s': %v", clusterKey, err)
		}

		if output.HasFlag() {
			err = output.Print(logForwarders)
			if err != nil {
				return fmt.Errorf("failed to output log forwarders: %v", err)
			}
			return nil
		}

		if len(logForwarders) == 0 {
			runtime.Reporter.Infof("There are no log forwarders configured for cluster '%s'", clusterKey)
			return nil
		}

		writer := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)

		fmt.Fprintf(writer, "ID\tTYPE\tSTATUS\n")
		for _, logForwarder := range logForwarders {
			logType := "Unknown"
			if logForwarder.S3() != nil {
				logType = "S3"
			} else if logForwarder.Cloudwatch() != nil {
				logType = "CloudWatch"
			}

			status := "N/A"
			if logForwarder.Status() != nil && logForwarder.Status().State() != "" {
				status = logForwarder.Status().State()
			}

			fmt.Fprintf(writer, "%s\t%s\t%s\n",
				logForwarder.ID(),
				logType,
				status,
			)
		}
		writer.Flush()
		return nil
	}
}
