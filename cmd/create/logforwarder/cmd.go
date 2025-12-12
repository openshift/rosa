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

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
	interactiveLogForwarding "github.com/openshift/rosa/pkg/interactive/logforwarding"
	"github.com/openshift/rosa/pkg/logforwarding"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use   = "log-forwarder -c <cluster-id>"
	short = "Create a log forwarder for a Hosted Control Plane cluster"
	long  = "Create a log forwarder to forward logs from a hosted cluster to external services " +
		"such as S3 or CloudWatch. Must create for an existing Hosted Control Plane cluster"
	example = `  # Create a log forwarder using a config file
  rosa create log-forwarder -c mycluster-hcp --log-fwd-config=s3.yml
  
  # Create a log forwarder interactively
  rosa create log-forwarder -c mycluster-hcp --interactive`
)

var aliases = []string{"logforwarder", "log-forwarder"}

func NewCreateLogForwarderCommand() *cobra.Command {
	options := NewCreateLogForwarderUserOptions()
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Aliases: aliases,
		Example: example,
		Args:    cobra.NoArgs,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), CreateLogForwarderRunner(options)),
	}

	flags := cmd.Flags()
	flags.StringVar(
		&options.logFwdConfig,
		logforwarding.FlagName,
		"",
		logforwarding.LogFwdConfigHelpMessage,
	)

	ocm.AddClusterFlag(cmd)
	interactive.AddFlag(flags)
	return cmd
}

func CreateLogForwarderRunner(userOptions *CreateLogForwarderUserOptions) rosa.CommandRunner {
	return func(_ context.Context, r *rosa.Runtime, cmd *cobra.Command, _ []string) error {
		options := NewCreateLogForwarderOptions()

		err := options.Bind(userOptions)
		if err != nil {
			return err
		}

		clusterKey := r.GetClusterKey()
		cluster, err := r.OCMClient.GetCluster(clusterKey, r.Creator)
		if err != nil {
			return err
		}

		if cluster.State() != cmv1.ClusterStateReady {
			return fmt.Errorf("cluster '%s' is not yet ready", clusterKey)
		}

		if !cluster.Hypershift().Enabled() {
			return fmt.Errorf("log forwarders are only supported for Hosted Control Plane clusters")
		}

		var logFwdS3ConfigObject *logforwarding.S3LogForwarderConfig
		var logFwdCloudWatchConfigObject *logforwarding.CloudWatchLogForwarderConfig

		if userOptions.logFwdConfig == "" {
			interactive.Enable()
		}

		if userOptions.logFwdConfig != "" {
			yamlObject, err := logforwarding.UnmarshalLogForwarderConfigYaml(userOptions.logFwdConfig)
			if err != nil {
				return fmt.Errorf("error parsing log forwarder config '%s': %v", userOptions.logFwdConfig, err)
			}
			if yamlObject.S3 != nil {
				logFwdS3ConfigObject = yamlObject.S3
			}
			if yamlObject.CloudWatch != nil {
				logFwdCloudWatchConfigObject = yamlObject.CloudWatch
			}
		} else if interactive.Enabled() {
			interactiveObject, err := interactiveLogForwarding.InteractiveLogForwardingConfig(r.OCMClient)
			if err != nil {
				return fmt.Errorf("failed to create log forwarder config: %v", err)
			}
			if interactiveObject.S3 != nil && interactiveObject.S3.S3ConfigBucketName != "" {
				logFwdS3ConfigObject = interactiveObject.S3
			}
			if interactiveObject.CloudWatch != nil && interactiveObject.CloudWatch.CloudWatchLogRoleArn != "" {
				logFwdCloudWatchConfigObject = interactiveObject.CloudWatch
			}
		}

		var logForwarderBuilder *cmv1.LogForwarderBuilder
		if logFwdS3ConfigObject != nil {
			logForwarderBuilder = logforwarding.BindS3LogForwarder(logFwdS3ConfigObject)
		} else if logFwdCloudWatchConfigObject != nil {
			logForwarderBuilder = logforwarding.BindCloudWatchLogForwarder(logFwdCloudWatchConfigObject)
		} else {
			return fmt.Errorf("no proper log forwarding configuration provided")
		}

		logForwarder, err := logForwarderBuilder.Build()
		if err != nil {
			return fmt.Errorf("failed to build log forwarder from inputs: %v", err)
		}

		createdLogForwarder, err := r.OCMClient.SetLogForwarder(cluster.ID(), logForwarder)
		if err != nil {
			return fmt.Errorf("failed to create log forwarder: %v", err)
		}

		if output.HasFlag() {
			err = output.Print(createdLogForwarder)
			if err != nil {
				return fmt.Errorf("failed to output log forwarder: %v", err)
			}
			return nil
		}

		r.Reporter.Infof("Successfully created log forwarder for HCP cluster '%s'", clusterKey)
		return nil
	}
}
