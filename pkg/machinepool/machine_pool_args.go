package machinepool

type MachinePoolArgs struct {
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
	MaxSurge              string
	MaxUnavailable        string
}
