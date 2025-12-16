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

package logforwarder

import (
	"context"
	"fmt"
	"regexp"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/fedramp"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "log-forwarder -c <cluster-id> <log-forwarder-id>"
	short   = "Delete log forwarder"
	long    = "Delete a log forwarder from a cluster."
	example = `  # Delete log forwarder with ID 'example-id' from a cluster named 'mycluster-hcp'
  rosa delete log-forwarder --cluster=mycluster-hcp example-id`
)

var (
	aliases = []string{"log_forwarder", "log-forwarder", "logforwarder"}
)

var logForwarderKeyRE = regexp.MustCompile(`^[a-z-0-9]+$`)

func NewDeleteLogForwarderCommand() *cobra.Command {
	options := NewDeleteLogForwarderUserOptions()
	cmd := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   short,
		Long:    long,
		Example: example,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), DeleteLogForwarderRunner(options)),
		Args:    cobra.MaximumNArgs(1),
	}

	flags := cmd.Flags()
	flags.StringVar(
		&options.logForwarder,
		"log-forwarder",
		"",
		"Log forwarder ID to delete",
	)

	ocm.AddClusterFlag(cmd)
	confirm.AddFlag(flags)
	return cmd
}

func DeleteLogForwarderRunner(userOptions *DeleteLogForwarderUserOptions) rosa.CommandRunner {
	return func(_ context.Context, runtime *rosa.Runtime, cmd *cobra.Command, argv []string) error {
		options := NewDeleteLogForwarderOptions()

		if fedramp.Enabled() {
			return fmt.Errorf("log forwarding is not supported on Govcloud")
		}

		err := options.Bind(userOptions, argv)
		if err != nil {
			return err
		}

		clusterKey := runtime.GetClusterKey()
		cluster, err := runtime.OCMClient.GetCluster(clusterKey, runtime.Creator)
		if err != nil {
			return err
		}
		if cluster.State() != cmv1.ClusterStateReady {
			return fmt.Errorf("cluster '%s' is not yet ready", clusterKey)
		}

		logForwarder, err := runtime.OCMClient.GetLogForwarderByID(cluster.ID(), options.LogForwarder())
		if err != nil {
			return fmt.Errorf("failed to get log forwarder '%s': %v", options.LogForwarder(), err)
		}
		if logForwarder == nil {
			return fmt.Errorf("log forwarder '%s' not found", options.LogForwarder())
		}

		if !confirm.Confirm("delete log forwarder '%s'?", options.LogForwarder()) {
			return nil
		}

		runtime.Reporter.Debugf("Deleting log forwarder '%s' for cluster '%s'", options.LogForwarder(),
			clusterKey)

		err = runtime.OCMClient.DeleteLogForwarder(cluster.ID(), options.LogForwarder())
		if err != nil {
			return fmt.Errorf("failed to delete log forwarder '%s': %v", options.LogForwarder(), err)
		}

		runtime.Reporter.Infof("Successfully deleted log forwarder '%s' from cluster '%s'",
			options.LogForwarder(), clusterKey)
		return nil
	}
}
