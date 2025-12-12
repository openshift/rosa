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
	"os"

	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
	errors "github.com/zgalor/weberr"

	interactiveLogForwarding "github.com/openshift/rosa/pkg/interactive/logforwarding"
	"github.com/openshift/rosa/pkg/logforwarding"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	short = "Edit a log forwarder for a cluster"
	long  = "Edit a log forwarder configuration for a cluster. A cluster ID must be provided, as well as " +
		"a valid log forwarder ID on that cluster. For example: 'rosa edit log-forwarder -c my-cluster-1 " +
		"2n4b8f8ai80cs6kmjmdgqlqplh73r411'"
)

var (
	logFwdConfig string
)

func NewEditLogForwarderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log-forwarder -c <cluster-id> <log-fwd-id> --log-fwd-config <path-to-config>",
		Short: short,
		Long:  long,
		Args:  cobra.ExactArgs(1),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	ocm.AddClusterFlag(cmd)
	flags.StringVar(
		&logFwdConfig,
		"log-fwd-config",
		"",
		"Path to YAML file containing log forwarder configuration",
	)

	cmd.Run = rosa.DefaultRunner(rosa.RuntimeWithOCM(), EditLogForwarderRunner)
	return cmd
}

func EditLogForwarderRunner(ctx context.Context, r *rosa.Runtime, command *cobra.Command, argv []string) error {
	if len(argv) != 1 {
		return fmt.Errorf("expected exactly one argument: log-forwarder ID")
	}

	logFwdID := argv[0]

	clusterKey := r.GetClusterKey()
	cluster, err := r.OCMClient.GetCluster(clusterKey, r.Creator)
	if err != nil {
		return err
	}

	configData := []byte{}

	if logFwdConfig != "" {
		configData, err = os.ReadFile(logFwdConfig)
		if err != nil {
			return fmt.Errorf("failed to read log forwarder config file '%s': %v", logFwdConfig, err)
		}
	}

	var logForwarderYaml logforwarding.LogForwarderYaml

	if len(configData) == 0 {
		interactiveObject, err := interactiveLogForwarding.InteractiveLogForwardingConfig(
			r.OCMClient)
		if err != nil {
			return errors.UserErrorf("failed to create log fowarder config: %s", err)
		}
		logForwarderYaml.S3 = interactiveObject.S3
		logForwarderYaml.CloudWatch = interactiveObject.CloudWatch
	} else {
		configDataString := string(configData)
		if configDataString == "{}" {
			return errors.UserErrorf("log forwarding config provided contained no valid log forwarders")
		}

		err := yaml.Unmarshal(configData, &logForwarderYaml)
		if err != nil {
			return err
		}
	}

	err = r.OCMClient.EditLogForwarder(cluster.ID(), logFwdID, logForwarderYaml)
	if err != nil {
		return fmt.Errorf("failed to edit log forwarder '%s' for cluster '%s': %s", logFwdID, cluster.ID(), err)
	}

	r.Reporter.Infof("Successfully edited log forwarder '%s' for cluster '%s'", logFwdID, cluster.ID())
	return nil
}
