package machinepool

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/securitygroups"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/reporter"
)

type CreateMachinepoolUserOptions struct {
	Name                  string
	InstanceType          string
	Replicas              int
	AutoscalingEnabled    bool
	MinReplicas           int
	MaxReplicas           int
	Labels                string
	Taints                string
	UseSpotInstances      bool
	SpotMaxPrice          string
	MultiAvailabilityZone bool
	AvailabilityZone      string
	Subnet                string
	Version               string
	Autorepair            bool
	TuningConfigs         string
	KubeletConfigs        string
	RootDiskSize          string
	SecurityGroupIds      []string
	NodeDrainGracePeriod  string
	Tags                  []string
}

const (
	use     = "machinepool"
	short   = "Add machine pool to cluster"
	long    = "Add a machine pool to the cluster."
	example = `  # Interactively add a machine pool to a cluster named "mycluster"
  rosa create machinepool --cluster=mycluster --interactive
  # Add a machine pool mp-1 with 3 replicas of m5.xlarge to a cluster
  rosa create machinepool --cluster=mycluster --name=mp-1 --replicas=3 --instance-type=m5.xlarge
  # Add a machine pool mp-1 with autoscaling enabled and 3 to 6 replicas of m5.xlarge to a cluster
  rosa create machinepool --cluster=mycluster --name=mp-1 --enable-autoscaling \
	--min-replicas=3 --max-replicas=6 --instance-type=m5.xlarge
  # Add a machine pool with labels to a cluster
  rosa create machinepool -c mycluster --name=mp-1 --replicas=2 --instance-type=r5.2xlarge --labels=foo=bar,bar=baz,
  # Add a machine pool with spot instances to a cluster
  rosa create machinepool -c mycluster --name=mp-1 --replicas=2 --instance-type=r5.2xlarge --use-spot-instances \
    --spot-max-price=0.5
  # Add a machine pool to a cluster and set the node drain grace period
  rosa create machinepool -c mycluster --name=mp-1 --node-drain-grace-period="90 minutes"`
)

type CreateMachinepoolOptions struct {
	reporter *reporter.Object

	args *CreateMachinepoolUserOptions
}

func NewCreateMachinepoolUserOptions() *CreateMachinepoolUserOptions {
	return &CreateMachinepoolUserOptions{}
}

func NewCreateMachinepoolOptions() *CreateMachinepoolOptions {
	return &CreateMachinepoolOptions{
		reporter: reporter.CreateReporter(),
		args:     &CreateMachinepoolUserOptions{},
	}
}

func (m *CreateMachinepoolOptions) Machinepool() *CreateMachinepoolUserOptions {
	return m.args
}

func (m *CreateMachinepoolOptions) Bind(args *CreateMachinepoolUserOptions, argv []string) error {
	m.args = args
	if len(argv) > 0 {
		m.args.Name = argv[0]
	}
	return nil
}

func BuildMachinePoolCreateCommandWithOptions() (*cobra.Command, *CreateMachinepoolUserOptions) {
	options := NewCreateMachinepoolUserOptions()
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Aliases: []string{"machinepools", "machine-pool", "machine-pools"},
		Example: example,
		Args:    cobra.NoArgs,
	}

	flags := cmd.Flags()
	ocm.AddClusterFlag(cmd)
	flags.StringVar(
		&options.Name,
		"name",
		"",
		"Name for the machine pool (required).",
	)

	flags.IntVar(
		&options.Replicas,
		"replicas",
		0,
		"Count of machines for the machine pool (required when autoscaling is disabled).",
	)

	flags.BoolVar(
		&options.AutoscalingEnabled,
		"enable-autoscaling",
		false,
		"Enable autoscaling for the machine pool.",
	)

	flags.IntVar(
		&options.MinReplicas,
		"min-replicas",
		0,
		"Minimum number of machines for the machine pool.",
	)

	flags.IntVar(
		&options.MaxReplicas,
		"max-replicas",
		0,
		"Maximum number of machines for the machine pool.",
	)

	flags.StringVar(
		&options.InstanceType,
		"instance-type",
		"m5.xlarge",
		"Instance type that should be used.",
	)

	flags.StringVar(
		&options.Labels,
		"labels",
		"",
		"Labels for machine pool. Format should be a comma-separated list of 'key=value'. "+
			"This list will overwrite any modifications made to Node labels on an ongoing basis.",
	)

	flags.StringVar(
		&options.Taints,
		"taints",
		"",
		"Taints for machine pool. Format should be a comma-separated list of 'key=value:ScheduleType'. "+
			"This list will overwrite any modifications made to Node taints on an ongoing basis.",
	)

	flags.BoolVar(
		&options.UseSpotInstances,
		"use-spot-instances",
		false,
		"Use spot instances for the machine pool.",
	)

	flags.StringVar(
		&options.SpotMaxPrice,
		"spot-max-price",
		"on-demand",
		"Max price for spot instance. If empty use the on-demand price.",
	)

	flags.BoolVar(
		&options.MultiAvailabilityZone,
		"multi-availability-zone",
		true,
		"Create a multi-AZ machine pool for a multi-AZ cluster")

	flags.StringVar(
		&options.AvailabilityZone,
		"availability-zone",
		"",
		"Select availability zone to create a single AZ machine pool for a multi-AZ cluster")

	flags.StringVar(
		&options.Subnet,
		"subnet",
		"",
		"Select subnet to create a single AZ machine pool for BYOVPC cluster")

	flags.StringVar(
		&options.Version,
		"version",
		"",
		"Version of OpenShift that will be used to install a machine pool for a hosted cluster,"+
			" for example \"4.12.4\"",
	)

	flags.BoolVar(
		&options.Autorepair,
		"autorepair",
		true,
		"Select auto-repair behaviour for a machinepool in a hosted cluster.",
	)

	flags.StringVar(
		&options.TuningConfigs,
		"tuning-configs",
		"",
		"Name of the tuning configs to be applied to the machine pool. Format should be a comma-separated list. "+
			"Tuning config must already exist. "+
			"This list will overwrite any modifications made to node tuning configs on an ongoing basis.",
	)

	flags.StringVar(
		&options.KubeletConfigs,
		"kubelet-configs",
		"",
		"Name of the kubelet config to be applied to the machine pool. A single kubelet config is allowed. "+
			"Kubelet config must already exist. "+
			"This will overwrite any modifications made to node kubelet configs on an ongoing basis.",
	)

	flags.StringVar(&options.RootDiskSize,
		"disk-size",
		"",
		"Root disk size with a suffix like GiB or TiB",
	)

	flags.StringSliceVar(&options.SecurityGroupIds,
		securitygroups.MachinePoolSecurityGroupFlag,
		nil,
		"The additional Security Group IDs to be added to the machine pool. "+
			"Format should be a comma-separated list.",
	)

	flags.StringVar(&options.NodeDrainGracePeriod,
		"node-drain-grace-period",
		"",
		"You may set a grace period for how long Pod Disruption Budget-protected workloads will be "+
			"respected when the NodePool is being replaced or upgraded.\nAfter this grace period, all remaining workloads "+
			"will be forcibly evicted.\n"+
			"Valid value is from 0 to 1 week (10080 minutes), and the supported units are 'minute|minutes' or "+
			"'hour|hours'. 0 or empty value means that the NodePool can be drained without any time limitations.\n"+
			"This flag is only supported for Hosted Control Planes.",
	)

	flags.StringSliceVar(
		&options.Tags,
		"tags",
		nil,
		"Apply user defined tags to all resources created by ROSA in AWS. "+
			"Tags are comma separated, for example: 'key value, foo bar'",
	)

	output.AddFlag(cmd)
	interactive.AddFlag(flags)
	return cmd, options
}
