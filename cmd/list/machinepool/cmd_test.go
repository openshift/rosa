package machinepool

import (
	"context"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	. "github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/test"
)

const (
	nodePoolName         = "nodepool85"
	clusterId            = "24vf9iitg3p6tlml88iml6j6mu095mh8"
	singleNodePoolOutput = "ID          AUTOSCALING  REPLICAS  INSTANCE TYPE  LABELS    TAINTS    AVAILABILITY ZONE  SUBNET  DISK SIZE  VERSION  AUTOREPAIR  \n" +
		"nodepool85  No           /0        m5.xlarge                          us-east-1a                 default    4.12.24  No          \n"

	singleMachinePoolOutput = "ID          AUTOSCALING  REPLICAS  INSTANCE TYPE  AVAILABILITY ZONES                  SPOT INSTANCES  DISK SIZE\n" +
		"nodepool85  No           0         m5.xlarge      us-east-1a, us-east-1b, us-east-1c  Yes (max $5)    default\n"

	multipleMachinePoolOutput = "ID           AUTOSCALING  REPLICAS  INSTANCE TYPE  LABELS      TAINTS       AVAILABILITY ZONES                  SPOT INSTANCES  DISK SIZE\n" +
		"nodepool85   No           0         m5.xlarge                               us-east-1a, us-east-1b, us-east-1c  Yes (max $5)    default\n" +
		"nodepool852  No           0         m5.xlarge      test=label               us-east-1a, us-east-1b, us-east-1c  Yes (max $5)    default\n" +
		"nodepool853  Yes          1-100     m5.xlarge      test=label  test=taint:  us-east-1a, us-east-1b, us-east-1c  Yes (max $5)    default\n"

	multipleNodePoolsOutput = "ID           AUTOSCALING  REPLICAS   INSTANCE TYPE  LABELS        TAINTS    AVAILABILITY ZONE  SUBNET  DISK SIZE  VERSION  AUTOREPAIR  \n" +
		"nodepool85   No           /0         m5.xlarge                              us-east-1a                 default    4.12.24  No          \n" +
		"nodepool852  Yes          /100-1000  m5.xlarge      test=label              us-east-1a                 default    4.12.24  No          \n"
)

var _ = Describe("List machine pool", func() {
	Context("List machine pool command", func() {
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

		mpResponse := formatMachinePool()
		multipleMpResponse := formatMachinePools()

		nodePoolResponse := formatNodePool()
		multipleNpResponse := formatNodePools()

		var t *test.TestingRuntime

		BeforeEach(func() {
			t = test.NewTestRuntime()
			SetOutput("")
		})
		Context("Hypershift", func() {
			It("Lists nodepool in hypershift cluster", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				runner := ListMachinePoolRunner()
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(singleNodePoolOutput))
			})
			It("Lists multiple nodepools in hypershift cluster", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, multipleNpResponse))
				runner := ListMachinePoolRunner()
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(multipleNodePoolsOutput))
			})
		})
		Context("ROSA Classic", func() {
			It("Lists machinepool in classic cluster", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponse))
				runner := ListMachinePoolRunner()
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(singleMachinePoolOutput))
			})
			It("Lists multiple machinepools in classic cluster", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, multipleMpResponse))
				runner := ListMachinePoolRunner()
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(multipleMachinePoolOutput))
			})
		})
		Context("Flag behavior tests", func() {
			It("Shows all columns when --all flag is used", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponse))
				runner := ListMachinePoolRunner()
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = cmd.Flag("all").Value.Set("true")
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())

				// With --all, should include columns like SUBNETS and SG IDS even if empty
				Expect(stdout).To(ContainSubstring("SUBNETS"))
				Expect(stdout).To(ContainSubstring("SG IDS"))
				Expect(stdout).To(ContainSubstring("AZ TYPE"))
				Expect(stdout).To(ContainSubstring("WIN-LI ENABLED"))
				Expect(stdout).To(ContainSubstring("DEDICATED HOST"))
			})

			It("Shows AZ TYPE column when --az-type flag is used", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponse))
				runner := ListMachinePoolRunner()
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = cmd.Flag("az-type").Value.Set("true")
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())

				// Should include AZ TYPE column
				Expect(stdout).To(ContainSubstring("AZ TYPE"))
			})

			It("Shows DEDICATED HOST column when --dedicated-host flag is used", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponse))
				runner := ListMachinePoolRunner()
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = cmd.Flag("dedicated-host").Value.Set("true")
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())

				// Should include DEDICATED HOST column
				Expect(stdout).To(ContainSubstring("DEDICATED HOST"))
			})

			It("Shows WIN-LI ENABLED column when --win-li flag is used", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponse))
				runner := ListMachinePoolRunner()
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = cmd.Flag("win-li").Value.Set("true")
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())

				// Should include WIN-LI ENABLED column
				Expect(stdout).To(ContainSubstring("WIN-LI ENABLED"))
			})

			It("Combines multiple flags correctly", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponse))
				runner := ListMachinePoolRunner()
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = cmd.Flag("az-type").Value.Set("true")
				Expect(err).ToNot(HaveOccurred())
				err = cmd.Flag("dedicated-host").Value.Set("true")
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())

				// Should include both columns
				Expect(stdout).To(ContainSubstring("AZ TYPE"))
				Expect(stdout).To(ContainSubstring("DEDICATED HOST"))
				// Should not include WIN-LI column since flag wasn't set
				Expect(stdout).ToNot(ContainSubstring("WIN-LI ENABLED"))
			})
		})

		Context("Column visibility tests", func() {
			It("Shows SUBNETS column when machine pools have subnet data", func() {
				awsMachinePoolPool := cmv1.NewAWSMachinePool().SpotMarketOptions(cmv1.NewAWSSpotMarketOptions().MaxPrice(5))
				machinePool, err := cmv1.NewMachinePool().ID(nodePoolName).AWS(awsMachinePoolPool).InstanceType("m5.xlarge").
					AvailabilityZones("us-east-1a", "us-east-1b", "us-east-1c").
					Subnets("subnet-1", "subnet-2").Build()
				Expect(err).ToNot(HaveOccurred())
				mpResponseWithSubnets := fmt.Sprintf("{\n  \"items\": [\n    %s\n  ],\n  \"page\": 0,\n  \"size\": 1,\n  \"total\": 1\n}",
					test.FormatResource(machinePool))

				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponseWithSubnets))
				runner := ListMachinePoolRunner()
				err = t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())

				// Should include SUBNETS column because machine pool has subnet data
				Expect(stdout).To(ContainSubstring("SUBNETS"))
				Expect(stdout).To(ContainSubstring("subnet-1, subnet-2"))
			})

			It("Shows SG IDS column when machine pools have security group data", func() {
				awsMachinePoolPool := cmv1.NewAWSMachinePool().
					SpotMarketOptions(cmv1.NewAWSSpotMarketOptions().MaxPrice(5)).
					AdditionalSecurityGroupIds("sg-1", "sg-2")
				machinePool, err := cmv1.NewMachinePool().ID(nodePoolName).AWS(awsMachinePoolPool).InstanceType("m5.xlarge").
					AvailabilityZones("us-east-1a", "us-east-1b", "us-east-1c").Build()
				Expect(err).ToNot(HaveOccurred())
				mpResponseWithSGs := fmt.Sprintf("{\n  \"items\": [\n    %s\n  ],\n  \"page\": 0,\n  \"size\": 1,\n  \"total\": 1\n}",
					test.FormatResource(machinePool))

				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponseWithSGs))
				runner := ListMachinePoolRunner()
				err = t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())

				// Should include SG IDS column because machine pool has security group data
				Expect(stdout).To(ContainSubstring("SG IDS"))
				Expect(stdout).To(ContainSubstring("sg-1, sg-2"))
			})

			It("Hides empty columns by default", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponse))
				runner := ListMachinePoolRunner()
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())

				// Should not include empty columns like SUBNETS and SG IDS by default
				Expect(stdout).ToNot(ContainSubstring("SUBNETS"))
				Expect(stdout).ToNot(ContainSubstring("SG IDS"))
				Expect(stdout).ToNot(ContainSubstring("LABELS"))
				Expect(stdout).ToNot(ContainSubstring("TAINTS"))
			})

			It("Shows columns with mixed data (some empty, some not)", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, multipleMpResponse))
				runner := ListMachinePoolRunner()
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())

				// Should include LABELS and TAINTS columns because some machine pools have data
				Expect(stdout).To(ContainSubstring("LABELS"))
				Expect(stdout).To(ContainSubstring("TAINTS"))
			})
		})
	})
})

// formatNodePool simulates the output of APIs for a fake node pool list
func formatNodePool() string {
	version := cmv1.NewVersion().ID("4.12.24").RawID("openshift-4.12.24")
	awsNodePool := cmv1.NewAWSNodePool().InstanceType("m5.xlarge")
	nodeDrain := cmv1.NewValue().Value(1).Unit("minute")
	nodePool, err := cmv1.NewNodePool().ID(nodePoolName).Version(version).
		AWSNodePool(awsNodePool).AvailabilityZone("us-east-1a").NodeDrainGracePeriod(nodeDrain).Build()
	Expect(err).ToNot(HaveOccurred())
	return fmt.Sprintf("{\n  \"items\": [\n    %s\n  ],\n  \"page\": 0,\n  \"size\": 1,\n  \"total\": 1\n}",
		test.FormatResource(nodePool))
}

// formatNodePools simulates the output of APIs for a fake node pool list (multiple)
func formatNodePools() string {
	version := cmv1.NewVersion().ID("4.12.24").RawID("openshift-4.12.24")
	awsNodePool := cmv1.NewAWSNodePool().InstanceType("m5.xlarge")
	nodeDrain := cmv1.NewValue().Value(1).Unit("minute")
	nodePoolBuilder := cmv1.NewNodePool().ID(nodePoolName).Version(version).
		AWSNodePool(awsNodePool).AvailabilityZone("us-east-1a").NodeDrainGracePeriod(nodeDrain)
	np1, err := nodePoolBuilder.Build()
	Expect(err).ToNot(HaveOccurred())
	nodePoolBuilder = nodePoolBuilder.ID(nodePoolName + "2").Labels(map[string]string{"test": "label"}).
		Autoscaling(cmv1.NewNodePoolAutoscaling().ID("scaler").MinReplica(100).MaxReplica(1000))
	np2, err := nodePoolBuilder.Build()
	Expect(err).ToNot(HaveOccurred())
	return fmt.Sprintf("{\n  \"items\": [\n    %s,\n%s\n  ],\n  \"page\": 0,\n  \"size\": 1,\n  \"total\": 1\n}",
		test.FormatResource(np1), test.FormatResource(np2))
}

// formatMachinePool simulates the output of APIs for a fake machine pool list
func formatMachinePool() string {
	awsMachinePoolPool := cmv1.NewAWSMachinePool().SpotMarketOptions(cmv1.NewAWSSpotMarketOptions().MaxPrice(5))
	machinePool, err := cmv1.NewMachinePool().ID(nodePoolName).AWS(awsMachinePoolPool).InstanceType("m5.xlarge").
		AvailabilityZones("us-east-1a", "us-east-1b", "us-east-1c").Build()
	Expect(err).ToNot(HaveOccurred())
	return fmt.Sprintf("{\n  \"items\": [\n    %s\n  ],\n  \"page\": 0,\n  \"size\": 1,\n  \"total\": 1\n}",
		test.FormatResource(machinePool))
}

// formatMachinePools simulates the output of APIs for a fake machine pool list (multiple)
func formatMachinePools() string {
	awsMachinePoolPool := cmv1.NewAWSMachinePool().SpotMarketOptions(cmv1.NewAWSSpotMarketOptions().MaxPrice(5))
	machinePoolBuilder := cmv1.NewMachinePool().ID(nodePoolName).AWS(awsMachinePoolPool).InstanceType("m5.xlarge").
		AvailabilityZones("us-east-1a", "us-east-1b", "us-east-1c")
	mp1, err := machinePoolBuilder.Build()
	Expect(err).ToNot(HaveOccurred())
	machinePoolBuilder = machinePoolBuilder.ID(nodePoolName + "2").Labels(map[string]string{"test": "label"})
	mp2, err := machinePoolBuilder.Build()
	Expect(err).ToNot(HaveOccurred())
	machinePoolBuilder = machinePoolBuilder.ID(nodePoolName + "3").Taints(cmv1.NewTaint().Key("test").
		Value("taint")).Autoscaling(cmv1.NewMachinePoolAutoscaling().ID("scaler").MaxReplicas(100).
		MinReplicas(1))
	mp3, err := machinePoolBuilder.Build()
	Expect(err).ToNot(HaveOccurred())
	return fmt.Sprintf("{\n  \"items\": [\n    %s,\n%s,\n%s\n  ],\n  \"page\": 0,\n  \"size\": 3,\n  \"total\": 3\n}",
		test.FormatResource(mp1), test.FormatResource(mp2), test.FormatResource(mp3))
}
