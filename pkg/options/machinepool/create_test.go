package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive/securitygroups"
)

var _ = Describe("BuildMachinePoolCreateCommandWithOptions", func() {
	var (
		cmd     *cobra.Command
		options *CreateMachinepoolUserOptions
	)

	BeforeEach(func() {
		cmd, options = BuildMachinePoolCreateCommandWithOptions()
	})

	It("should create a command with the expected use, short, long, and example descriptions", func() {
		Expect(cmd.Use).To(Equal(use))
		Expect(cmd.Short).To(Equal(short))
		Expect(cmd.Long).To(Equal(long))
		Expect(cmd.Example).To(Equal(example))
	})

	It("should initialize options with default values", func() {
		Expect(options.Name).To(Equal(""))
		Expect(options.Replicas).To(Equal(0))
		Expect(options.AutoscalingEnabled).To(Equal(false))
		Expect(options.MinReplicas).To(Equal(0))
		Expect(options.MaxReplicas).To(Equal(0))
		Expect(options.InstanceType).To(Equal("m5.xlarge"))
		Expect(options.Labels).To(Equal(""))
		Expect(options.Taints).To(Equal(""))
		Expect(options.UseSpotInstances).To(Equal(false))
		Expect(options.SpotMaxPrice).To(Equal("on-demand"))
		Expect(options.MultiAvailabilityZone).To(Equal(true))
		Expect(options.AvailabilityZone).To(Equal(""))
		Expect(options.Subnet).To(Equal(""))
		Expect(options.Version).To(Equal(""))
		Expect(options.Autorepair).To(Equal(true))
		Expect(options.TuningConfigs).To(Equal(""))
		Expect(options.KubeletConfigs).To(Equal(""))
		Expect(options.RootDiskSize).To(Equal(""))
		Expect(options.SecurityGroupIds).To(BeNil())
		Expect(options.NodeDrainGracePeriod).To(Equal(""))
		Expect(options.Tags).To(BeNil())
		Expect(options.MaxSurge).To(Equal("1"))
		Expect(options.MaxUnavailable).To(Equal("0"))
	})

	It("should have flags set with the correct default values", func() {
		flags := cmd.Flags()

		name, _ := flags.GetString("name")
		Expect(name).To(Equal(""))

		replicas, _ := flags.GetInt("replicas")
		Expect(replicas).To(Equal(0))

		autoscalingEnabled, _ := flags.GetBool("enable-autoscaling")
		Expect(autoscalingEnabled).To(Equal(false))

		minReplicas, _ := flags.GetInt("min-replicas")
		Expect(minReplicas).To(Equal(0))

		maxReplicas, _ := flags.GetInt("max-replicas")
		Expect(maxReplicas).To(Equal(0))

		instanceType, _ := flags.GetString("instance-type")
		Expect(instanceType).To(Equal("m5.xlarge"))

		labels, _ := flags.GetString("labels")
		Expect(labels).To(Equal(""))

		taints, _ := flags.GetString("taints")
		Expect(taints).To(Equal(""))

		useSpotInstances, _ := flags.GetBool("use-spot-instances")
		Expect(useSpotInstances).To(Equal(false))

		spotMaxPrice, _ := flags.GetString("spot-max-price")
		Expect(spotMaxPrice).To(Equal("on-demand"))

		multiAZ, _ := flags.GetBool("multi-availability-zone")
		Expect(multiAZ).To(Equal(true))

		availabilityZone, _ := flags.GetString("availability-zone")
		Expect(availabilityZone).To(Equal(""))

		subnet, _ := flags.GetString("subnet")
		Expect(subnet).To(Equal(""))

		version, _ := flags.GetString("version")
		Expect(version).To(Equal(""))

		autorepair, _ := flags.GetBool("autorepair")
		Expect(autorepair).To(Equal(true))

		tuningConfigs, _ := flags.GetString("tuning-configs")
		Expect(tuningConfigs).To(Equal(""))

		kubeletConfigs, _ := flags.GetString("kubelet-configs")
		Expect(kubeletConfigs).To(Equal(""))

		rootDiskSize, _ := flags.GetString("disk-size")
		Expect(rootDiskSize).To(Equal(""))

		securityGroupIds, _ := flags.GetStringSlice(securitygroups.MachinePoolSecurityGroupFlag)
		Expect(securityGroupIds).To(BeEmpty())

		nodeDrainGracePeriod, _ := flags.GetString("node-drain-grace-period")
		Expect(nodeDrainGracePeriod).To(Equal(""))

		tags, _ := flags.GetStringSlice("tags")
		Expect(tags).To(BeEmpty())

		maxSurge, _ := flags.GetString("max-surge")
		Expect(maxSurge).To(Equal("1"))

		maxUnavailable, _ := flags.GetString("max-unavailable")
		Expect(maxUnavailable).To(Equal("0"))
	})
})
