package machinepool

import (
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/ocm/output"
	"github.com/openshift/rosa/pkg/test"
)

const (
	nodePoolName         = "nodepool85"
	describeStringOutput = `
ID:                                    nodepool85
Cluster ID:                            24vf9iitg3p6tlml88iml6j6mu095mh8
Autoscaling:                           No
Desired replicas:                      0
Current replicas:                      
Instance type:                         m5.xlarge
Labels:                                
Tags:                                  
Taints:                                
Availability zone:                     us-east-1a
Subnet:                                
Version:                               4.12.24
Autorepair:                            No
Tuning configs:                        
Additional security group IDs:         
Node drain grace period:               1 minute
Message:                               
`
	describeStringWithUpgradeOutput = `
ID:                                    nodepool85
Cluster ID:                            24vf9iitg3p6tlml88iml6j6mu095mh8
Autoscaling:                           No
Desired replicas:                      0
Current replicas:                      
Instance type:                         m5.xlarge
Labels:                                
Tags:                                  
Taints:                                
Availability zone:                     us-east-1a
Subnet:                                
Version:                               4.12.24
Autorepair:                            No
Tuning configs:                        
Additional security group IDs:         
Node drain grace period:               1 minute
Message:                               
Scheduled upgrade:                     scheduled 4.12.25 on 2023-08-07 15:22 UTC
`
	describeStringWithTagsOutput = `
ID:                                    nodepool85
Cluster ID:                            24vf9iitg3p6tlml88iml6j6mu095mh8
Autoscaling:                           No
Desired replicas:                      0
Current replicas:                      
Instance type:                         m5.xlarge
Labels:                                
Tags:                                  foo=bar
Taints:                                
Availability zone:                     us-east-1a
Subnet:                                
Version:                               4.12.24
Autorepair:                            No
Tuning configs:                        
Additional security group IDs:         
Node drain grace period:               1 minute
Message:                               
Scheduled upgrade:                     scheduled 4.12.25 on 2023-08-07 15:22 UTC
`

	describeYamlWithUpgradeOutput = `availability_zone: us-east-1a
aws_node_pool:
  instance_type: m5.xlarge
  kind: AWSNodePool
id: nodepool85
kind: NodePool
node_drain_grace_period:
  unit: minute
  value: 1
scheduledUpgrade:
  nextRun: 2023-08-07 15:22 UTC
  state: scheduled
  version: 4.12.25
version:
  id: 4.12.24
  kind: Version
  raw_id: openshift-4.12.24
`
	describeClassicStringOutput = `
ID:                                    nodepool85
Cluster ID:                            24vf9iitg3p6tlml88iml6j6mu095mh8
Autoscaling:                           No
Replicas:                              0
Instance type:                         m5.xlarge
Labels:                                
Taints:                                
Availability zones:                    us-east-1a, us-east-1b, us-east-1c
Subnets:                               
Spot instances:                        Yes (max $5)
Disk size:                             default
Additional Security Group IDs:         
`
	describeClassicYamlOutput = `availability_zones:
- us-east-1a
- us-east-1b
- us-east-1c
aws:
  kind: AWSMachinePool
  spot_market_options:
    kind: AWSSpotMarketOptions
    max_price: 5
id: nodepool85
instance_type: m5.xlarge
kind: MachinePool
`
)

var _ = Describe("Upgrade machine pool", func() {
	Context("Upgrade machine pool command", func() {
		// Full diff for long string to help debugging
		format.TruncatedDiff = false
		var testRuntime test.TestingRuntime

		mockClusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
		})

		hypershiftClusterReady := test.FormatClusterList([]*cmv1.Cluster{mockClusterReady})

		mockClassicClusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(false))
		})
		classicClusterReady := test.FormatClusterList([]*cmv1.Cluster{mockClassicClusterReady})

		nodePoolResponse := formatNodePool()
		npResponseAwsTags := formatNodePoolWithTags()
		mpResponse := formatMachinePool()

		upgradePolicies := make([]*cmv1.NodePoolUpgradePolicy, 0)
		upgradePolicies = append(upgradePolicies, buildNodePoolUpgradePolicy())
		nodePoolUpgradePolicy := test.FormatNodePoolUpgradePolicyList(upgradePolicies)

		noNodePoolUpgradePolicy := test.FormatNodePoolUpgradePolicyList([]*cmv1.NodePoolUpgradePolicy{})

		BeforeEach(func() {
			testRuntime.InitRuntime()
			// Reset flag to avoid any side effect on other tests
			Cmd.Flags().Set("output", "")
		})
		It("Fails if we are not specifying a machine pool name", func() {
			args.machinePool = ""
			_, _, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime, Cmd, &[]string{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("You need to specify a machine pool name"))
		})
		Context("Hypershift", func() {
			It("Pass a machine pool name through argv but it is not found", func() {
				args.machinePool = ""
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
				stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
					Cmd, &[]string{nodePoolName})
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Machine pool '%s' not found", nodePoolName)))
				Expect(stdout).To(BeEmpty())
				Expect(stderr).To(BeEmpty())
			})
			It("Pass a machine pool name through parameter but it is not found", func() {
				args.machinePool = nodePoolName
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
				stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
					Cmd, &[]string{})
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Machine pool '%s' not found", nodePoolName)))
				Expect(stdout).To(BeEmpty())
				Expect(stderr).To(BeEmpty())
			})
			It("Pass a machine pool name through parameter and it is found. no upgrades", func() {
				args.machinePool = nodePoolName
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				// First get
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				// Second get for upgrades
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, noNodePoolUpgradePolicy))
				stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
					Cmd, &[]string{})
				Expect(err).To(BeNil())
				Expect(stdout).To(Equal(describeStringOutput))
				Expect(stderr).To(BeEmpty())
			})
			It("Pass a machine pool name through parameter and it is found. 1 upgrade", func() {
				args.machinePool = nodePoolName
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				// First get
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				// Second get for upgrades
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolUpgradePolicy))
				stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
					Cmd, &[]string{})
				Expect(err).To(BeNil())
				Expect(stdout).To(Equal(describeStringWithUpgradeOutput))
				Expect(stderr).To(BeEmpty())
			})
			It("Pass a machine pool name through parameter and it is found. 1 upgrade. Yaml output", func() {
				args.machinePool = nodePoolName
				Cmd.Flags().Set("output", "yaml")
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				// First get
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				// Second get for upgrades
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolUpgradePolicy))
				stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
					Cmd, &[]string{})
				Expect(err).To(BeNil())
				Expect(stdout).To(Equal(describeYamlWithUpgradeOutput))
				Expect(stderr).To(BeEmpty())
			})
			It("Pass a machine pool name through parameter and it is found. Has AWS tags", func() {
				args.machinePool = nodePoolName
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				// First get
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, npResponseAwsTags))
				// Second get for upgrades
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, npResponseAwsTags))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolUpgradePolicy))
				stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
					Cmd, &[]string{})
				Expect(err).To(BeNil())
				Expect(stdout).To(Equal(describeStringWithTagsOutput))
				Expect(stderr).To(BeEmpty())
			})
		})
		Context("ROSA Classic", func() {
			It("Pass a machine pool name through argv but it is not found", func() {
				args.machinePool = ""
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
				stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
					Cmd, &[]string{nodePoolName})
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Machine pool '%s' not found", nodePoolName)))
				Expect(stdout).To(BeEmpty())
				Expect(stderr).To(BeEmpty())
			})
			It("Pass a machine pool name through parameter but it is not found", func() {
				args.machinePool = nodePoolName
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
				stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
					Cmd, &[]string{})
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Machine pool '%s' not found", nodePoolName)))
				Expect(stdout).To(BeEmpty())
				Expect(stderr).To(BeEmpty())
			})
			It("Pass a machine pool name through parameter and it is found", func() {
				args.machinePool = nodePoolName
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponse))
				stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
					Cmd, &[]string{})
				Expect(err).To(BeNil())
				Expect(stdout).To(Equal(describeClassicStringOutput))
				Expect(stderr).To(BeEmpty())
			})
			It("Format AWS additional security groups if exist", func() {
				securityGroupsIds := []string{"123", "321"}
				awsNodePool, err := cmv1.NewAWSNodePool().AdditionalSecurityGroupIds(securityGroupsIds...).Build()
				Expect(err).ToNot(HaveOccurred())

				securityGroupsOutput := output.PrintNodePoolAdditionalSecurityGroups(awsNodePool)
				Expect(securityGroupsOutput).To(Equal("123, 321"))
			})
			It("Return an empty list for additional security groups if empty AWS node pool is passed", func() {
				awsNodePool, err := cmv1.NewAWSNodePool().Build()
				Expect(err).ToNot(HaveOccurred())

				securityGroupsOutput := output.PrintNodePoolAdditionalSecurityGroups(awsNodePool)
				Expect(securityGroupsOutput).To(Equal(""))
			})
			It("Pass a machine pool name through parameter and it is found. yaml output", func() {
				args.machinePool = nodePoolName
				Cmd.Flags().Set("output", "yaml")
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponse))
				stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
					Cmd, &[]string{})
				Expect(err).To(BeNil())
				Expect(stdout).To(Equal(describeClassicYamlOutput))
				Expect(stderr).To(BeEmpty())
			})
		})
	})
})

// formatNodePool simulates the output of APIs for a fake node pool
func formatNodePool() string {
	version := cmv1.NewVersion().ID("4.12.24").RawID("openshift-4.12.24")
	awsNodePool := cmv1.NewAWSNodePool().InstanceType("m5.xlarge")
	nodeDrain := cmv1.NewValue().Value(1).Unit("minute")
	np, err := cmv1.NewNodePool().ID(nodePoolName).Version(version).
		AWSNodePool(awsNodePool).AvailabilityZone("us-east-1a").NodeDrainGracePeriod(nodeDrain).Build()
	Expect(err).To(BeNil())
	return test.FormatResource(np)
}

// formatNodePool simulates the output of APIs for a fake node pool with AWS tags
func formatNodePoolWithTags() string {
	version := cmv1.NewVersion().ID("4.12.24").RawID("openshift-4.12.24")
	awsNodePool := cmv1.NewAWSNodePool().InstanceType("m5.xlarge").Tags(map[string]string{"foo": "bar"})
	nodeDrain := cmv1.NewValue().Value(1).Unit("minute")
	np, err := cmv1.NewNodePool().ID(nodePoolName).Version(version).
		AWSNodePool(awsNodePool).AvailabilityZone("us-east-1a").NodeDrainGracePeriod(nodeDrain).Build()
	Expect(err).To(BeNil())
	return test.FormatResource(np)
}

// formatMachinePool simulates the output of APIs for a fake machine pool
func formatMachinePool() string {
	awsMachinePoolPool := cmv1.NewAWSMachinePool().SpotMarketOptions(cmv1.NewAWSSpotMarketOptions().MaxPrice(5))
	mp, err := cmv1.NewMachinePool().ID(nodePoolName).AWS(awsMachinePoolPool).InstanceType("m5.xlarge").
		AvailabilityZones("us-east-1a", "us-east-1b", "us-east-1c").Build()
	Expect(err).To(BeNil())
	return test.FormatResource(mp)
}

func buildNodePoolUpgradePolicy() *cmv1.NodePoolUpgradePolicy {
	t, err := time.Parse(time.RFC3339, "2023-08-07T15:22:00Z")
	Expect(err).To(BeNil())
	state := cmv1.NewUpgradePolicyState().Value(cmv1.UpgradePolicyStateValueScheduled)
	policy, err := cmv1.NewNodePoolUpgradePolicy().ScheduleType(cmv1.ScheduleTypeManual).
		UpgradeType(cmv1.UpgradeTypeNodePool).Version("4.12.25").State(state).NextRun(t).Build()
	Expect(err).To(BeNil())
	return policy
}
