package logforwarders

import (
	"context"
	"fmt"
	"regexp"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/logforwarding"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "log-forwarder"
	short   = "Show details of a specific log forwarder used by a cluster"
	example = `rosa describe log-forwarder <log_fwd_id> -c mycluster-hcp`
)

var logForwarderKeyRE = regexp.MustCompile(`^[a-z0-9]+$`)

func NewDescribeLogForwarderCommand() *cobra.Command {
	options := NewDescribeLogForwarderUserOptions()
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Example: example,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), DescribeLogForwarderRunner(options)),
		Args:    cobra.MaximumNArgs(1),
	}

	flags := cmd.Flags()
	flags.StringVar(
		&options.logForwarder,
		"log-forwarder",
		"",
		"Log forwarder ID of the cluster to target",
	)

	ocm.AddClusterFlag(cmd)
	output.AddFlag(cmd)
	return cmd
}

func DescribeLogForwarderRunner(userOptions DescribeLogForwarderUserOptions) rosa.CommandRunner {
	return func(_ context.Context, runtime *rosa.Runtime, cmd *cobra.Command, argv []string) error {
		options := NewDescribeLogForwarderOptions()
		if len(argv) == 1 && !cmd.Flag("log-forwarder").Changed {
			userOptions.logForwarder = argv[0]
		} else {
			err := cmd.ParseFlags(argv)
			if err != nil {
				return fmt.Errorf("unable to parse flags: %v", err)
			}
			userOptions.logForwarder = cmd.Flag("log-forwarder").Value.String()
		}
		err := options.Bind(userOptions)
		if err != nil {
			return err
		}
		clusterKey := runtime.GetClusterKey()
		cluster := runtime.FetchCluster()
		if cluster.State() != cmv1.ClusterStateReady {
			return fmt.Errorf("cluster '%s' is not yet ready", clusterKey)
		}
		logForwarder, err := runtime.OCMClient.GetLogForwarderByID(cluster.ID(), options.args.logForwarder)
		if err != nil {
			return fmt.Errorf("failed to get log forwarder '%s': %v", options.args.logForwarder, err)
		}
		if logForwarder == nil {
			return fmt.Errorf("failed to get log forwarder '%s'", options.args.logForwarder)
		}

		if output.HasFlag() {
			err = output.Print(logForwarder)
			if err != nil {
				return fmt.Errorf("failed to output log forwarder '%s': %v", options.args.logForwarder, err)
			}
			return nil
		}

		fmt.Print(logforwarding.LogForwarderObjectAsString(logForwarder))
		return nil
	}
}
