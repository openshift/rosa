package machinepool

import (
	"context"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/ocm/output"
	. "github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/test"
)

const (
	nodePoolName         = "nodepool85"
	clusterId            = "24vf9iitg3p6tlml88iml6j6mu095mh8"
	describeStringOutput = `
ID:                                    nodepool85
Cluster ID:                            24vf9iitg3p6tlml88iml6j6mu095mh8
Autoscaling:                           No
Desired replicas:                      0
Current replicas:                      
Instance type:                         m5.xlarge
Image type:                            
Labels:                                
Tags:                                  
Taints:                                
Availability zone:                     us-east-1a
Subnet:                                
Disk Size:                             300 GiB
Version:                               4.12.24
EC2 Metadata Http Tokens:              optional
Autorepair:                            No
Tuning configs:                        
Kubelet configs:                       
Additional security group IDs:         
Node drain grace period:               1 minute
Capacity Reservation:                  
Management upgrade:                    
 - Type:                               Replace
 - Max surge:                          1
 - Max unavailable:                    0
Message:                               
`
	describeStringWithCapacityReservationOutput = `
ID:                                    nodepool85
Cluster ID:                            24vf9iitg3p6tlml88iml6j6mu095mh8
Autoscaling:                           No
Desired replicas:                      0
Current replicas:                      
Instance type:                         m5.xlarge
Image type:                            
Labels:                                
Tags:                                  
Taints:                                
Availability zone:                     us-east-1a
Subnet:                                
Disk Size:                             300 GiB
Version:                               4.12.24
EC2 Metadata Http Tokens:              optional
Autorepair:                            No
Tuning configs:                        
Kubelet configs:                       
Additional security group IDs:         
Node drain grace period:               1 minute
Capacity Reservation:                  
 - ID:                                 test-id
 - Type:                               OnDemand
Management upgrade:                    
 - Type:                               Replace
 - Max surge:                          1
 - Max unavailable:                    0
Message:                               
`
	describeStringWithUpgradeOutput = `
ID:                                    nodepool85
Cluster ID:                            24vf9iitg3p6tlml88iml6j6mu095mh8
Autoscaling:                           No
Desired replicas:                      0
Current replicas:                      
Instance type:                         m5.xlarge
Image type:                            
Labels:                                
Tags:                                  
Taints:                                
Availability zone:                     us-east-1a
Subnet:                                
Disk Size:                             300 GiB
Version:                               4.12.24
EC2 Metadata Http Tokens:              optional
Autorepair:                            No
Tuning configs:                        
Kubelet configs:                       
Additional security group IDs:         
Node drain grace period:               1 minute
Capacity Reservation:                  
Management upgrade:                    
 - Type:                               Replace
 - Max surge:                          1
 - Max unavailable:                    0
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
Image type:                            
Labels:                                
Tags:                                  foo=bar
Taints:                                
Availability zone:                     us-east-1a
Subnet:                                
Disk Size:                             300 GiB
Version:                               4.12.24
EC2 Metadata Http Tokens:              optional
Autorepair:                            No
Tuning configs:                        
Kubelet configs:                       
Additional security group IDs:         
Node drain grace period:               1 minute
Capacity Reservation:                  
Management upgrade:                    
 - Type:                               Replace
 - Max surge:                          1
 - Max unavailable:                    0
Message:                               
Scheduled upgrade:                     scheduled 4.12.25 on 2023-08-07 15:22 UTC
`

	describeYamlWithUpgradeOutput = `availability_zone: us-east-1a
aws_node_pool:
  instance_type: m5.xlarge
  kind: AWSNodePool
  root_volume:
    size: 300
id: nodepool85
kind: NodePool
management_upgrade:
  kind: NodePoolManagementUpgrade
  max_surge: "1"
  max_unavailable: "0"
  type: Replace
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
Tags:                                  
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
		npResponseCapacityReservation := formatNodePoolWithCapacityReservation()
		mpResponse := formatMachinePool()

		upgradePolicies := make([]*cmv1.NodePoolUpgradePolicy, 0)
		upgradePolicies = append(upgradePolicies, buildNodePoolUpgradePolicy())
		nodePoolUpgradePolicy := test.FormatNodePoolUpgradePolicyList(upgradePolicies)

		noNodePoolUpgradePolicy := test.FormatNodePoolUpgradePolicyList([]*cmv1.NodePoolUpgradePolicy{})

		var t *test.TestingRuntime

		BeforeEach(func() {
			t = test.NewTestRuntime()
			SetOutput("")
		})
		It("Fails if we are not specifying a machine pool name", func() {
			runner := DescribeMachinePoolRunner(NewDescribeMachinepoolUserOptions())
			err := runner(context.Background(), t.RosaRuntime, NewDescribeMachinePoolCommand(), []string{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("you need to specify a machine pool name"))
		})
		Context("Hypershift", func() {
			It("Works without passing `--machinepool`", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				// First get
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				// Second get for upgrades
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolUpgradePolicy))
				args := NewDescribeMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DescribeMachinePoolRunner(args)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDescribeMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{nodePoolName})
				Expect(err).To(BeNil())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(describeStringWithUpgradeOutput))
			})
			It("Pass a machine pool name through argv but it is not found", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
				args := NewDescribeMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DescribeMachinePoolRunner(args)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDescribeMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{"--machinepool", nodePoolName})
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Machine pool '%s' not found", nodePoolName)))
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(""))
			})
			It("Pass a machine pool name through parameter and it is found. no upgrades", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				// First get
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				// Second get for upgrades
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, noNodePoolUpgradePolicy))
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				args := NewDescribeMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DescribeMachinePoolRunner(args)
				cmd := NewDescribeMachinePoolCommand()
				cmd.Flag("cluster").Value.Set(clusterId)
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{"--machinepool", nodePoolName})
				Expect(err).To(BeNil())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(describeStringOutput))
			})
			It("Pass a machine pool name through parameter and it is found. 1 upgrade", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				// First get
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				// Second get for upgrades
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolUpgradePolicy))
				args := NewDescribeMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DescribeMachinePoolRunner(args)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDescribeMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{"--machinepool", nodePoolName})
				Expect(err).To(BeNil())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(describeStringWithUpgradeOutput))
			})
			It("Pass a machine pool name through parameter and it is found. 1 upgrade. Yaml output", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				// First get
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				// Second get for upgrades
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolUpgradePolicy))
				args := NewDescribeMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DescribeMachinePoolRunner(args)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDescribeMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = cmd.Flag("output").Value.Set("yaml")
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{"--machinepool", nodePoolName})
				Expect(err).To(BeNil())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(describeYamlWithUpgradeOutput))
			})
			It("Pass a machine pool name through parameter and it is found. Has AWS tags", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				// First get
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, npResponseAwsTags))
				// Second get for upgrades
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, npResponseAwsTags))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolUpgradePolicy))
				args := NewDescribeMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DescribeMachinePoolRunner(args)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDescribeMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{"--machinepool", nodePoolName})
				Expect(err).To(BeNil())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(describeStringWithTagsOutput))
			})
			It("Pass a machine pool name through parameter and it is found. Has Capacity Reservation", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				// First get
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, npResponseCapacityReservation))
				// Second get for upgrades
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, npResponseCapacityReservation))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, noNodePoolUpgradePolicy))
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				args := NewDescribeMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DescribeMachinePoolRunner(args)
				cmd := NewDescribeMachinePoolCommand()
				_ = cmd.Flag("cluster").Value.Set(clusterId)
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{"--machinepool", nodePoolName})
				Expect(err).To(BeNil())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(describeStringWithCapacityReservationOutput))
			})
		})
		Context("ROSA Classic", func() {
			It("Pass a machine pool name through argv but it is not found", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
				args := NewDescribeMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DescribeMachinePoolRunner(args)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDescribeMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{"--machinepool", nodePoolName})
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Machine pool '%s' not found", nodePoolName)))
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(""))
			})
			It("Pass a machine pool name through parameter but it is not found", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
				args := NewDescribeMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DescribeMachinePoolRunner(args)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDescribeMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{"--machinepool", nodePoolName})
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Machine pool '%s' not found", nodePoolName)))
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(""))
			})
			It("Pass a machine pool name through parameter and it is found", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponse))
				args := NewDescribeMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DescribeMachinePoolRunner(args)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDescribeMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{"--machinepool", nodePoolName})
				Expect(err).To(BeNil())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(describeClassicStringOutput))
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
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponse))
				args := NewDescribeMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DescribeMachinePoolRunner(args)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDescribeMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = cmd.Flag("output").Value.Set("yaml")
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{"--machinepool", nodePoolName})
				Expect(err).To(BeNil())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(describeClassicYamlOutput))
			})
		})
	})
})

// formatNodePool simulates the output of APIs for a fake node pool
func formatNodePool() string {
	version := cmv1.NewVersion().ID("4.12.24").RawID("openshift-4.12.24")
	awsNodePool := cmv1.NewAWSNodePool().InstanceType("m5.xlarge").RootVolume(cmv1.NewAWSVolume().Size(300))
	nodeDrain := cmv1.NewValue().Value(1).Unit("minute")
	mgmtUpgrade := cmv1.NewNodePoolManagementUpgrade().Type("Replace").MaxSurge("1").MaxUnavailable("0")
	np, err := cmv1.NewNodePool().ID(nodePoolName).Version(version).
		AWSNodePool(awsNodePool).AvailabilityZone("us-east-1a").NodeDrainGracePeriod(nodeDrain).
		ManagementUpgrade(mgmtUpgrade).Build()
	Expect(err).To(BeNil())
	return test.FormatResource(np)
}

// formatNodePoolWithTags simulates the output of APIs for a fake node pool with AWS tags
func formatNodePoolWithTags() string {
	version := cmv1.NewVersion().ID("4.12.24").RawID("openshift-4.12.24")
	awsNodePool := cmv1.NewAWSNodePool().InstanceType("m5.xlarge").Tags(map[string]string{"foo": "bar"}).
		RootVolume(cmv1.NewAWSVolume().Size(300))
	nodeDrain := cmv1.NewValue().Value(1).Unit("minute")
	mgmtUpgrade := cmv1.NewNodePoolManagementUpgrade().Type("Replace").MaxSurge("1").MaxUnavailable("0")
	np, err := cmv1.NewNodePool().ID(nodePoolName).Version(version).
		AWSNodePool(awsNodePool).AvailabilityZone("us-east-1a").NodeDrainGracePeriod(nodeDrain).
		ManagementUpgrade(mgmtUpgrade).Build()
	Expect(err).To(BeNil())
	return test.FormatResource(np)
}

// formatNodePoolWithCapacityReservation simulates the output of APIs for a fake node pool with a Capacity Reservation ID
func formatNodePoolWithCapacityReservation() string {
	version := cmv1.NewVersion().ID("4.12.24").RawID("openshift-4.12.24")
	awsNodePool := cmv1.NewAWSNodePool().InstanceType("m5.xlarge").RootVolume(cmv1.NewAWSVolume().
		Size(300)).CapacityReservation(cmv1.NewAWSCapacityReservation().Id("test-id").
		MarketType(cmv1.MarketTypeOnDemand))
	nodeDrain := cmv1.NewValue().Value(1).Unit("minute")
	mgmtUpgrade := cmv1.NewNodePoolManagementUpgrade().Type("Replace").MaxSurge("1").MaxUnavailable("0")
	np, err := cmv1.NewNodePool().ID(nodePoolName).Version(version).
		AWSNodePool(awsNodePool).AvailabilityZone("us-east-1a").NodeDrainGracePeriod(nodeDrain).
		ManagementUpgrade(mgmtUpgrade).Build()
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
