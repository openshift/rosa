package machinepool

import (
	"bytes"
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	. "github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/test"
)

var (
	nodePoolId = "test-nodepool"
)

var _ = Describe("Edit Machinepool", func() {
	Context("getNodePoolReplicas", func() {

		// Full diff for long string to help debugging
		format.TruncatedDiff = false

		clusterId := "classic-cluster"

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

		version := cmv1.NewVersion().ID("4.12.24").RawID("openshift-4.12.24")
		awsNodePool := cmv1.NewAWSNodePool().InstanceType("m5.xlarge")
		nodeDrain := cmv1.NewValue().Value(1).Unit("minute")
		nodePool, err := cmv1.NewNodePool().ID(nodePoolId).Version(version).
			AWSNodePool(awsNodePool).AvailabilityZone("us-east-1a").NodeDrainGracePeriod(nodeDrain).Build()
		Expect(err).ToNot(HaveOccurred())

		awsMachinePoolPool := cmv1.NewAWSMachinePool().SpotMarketOptions(cmv1.NewAWSSpotMarketOptions().MaxPrice(5))
		machinePool, err := cmv1.NewMachinePool().ID(nodePoolId).AWS(awsMachinePoolPool).InstanceType("m5.xlarge").
			AvailabilityZones("us-east-1a", "us-east-1b", "us-east-1c").Build()
		Expect(err).ToNot(HaveOccurred())

		nodePoolResponse := test.FormatNodePoolList([]*cmv1.NodePool{nodePool})
		mpResponse := test.FormatMachinePoolList([]*cmv1.MachinePool{machinePool})
		nodePoolAutoResponse := test.FormatNodePoolAutoscaling(nodePoolId)

		var t *test.TestingRuntime

		BeforeEach(func() {
			t = test.NewTestRuntime()
			SetOutput("")
		})
		AfterEach(func() {
			t.SetCluster("", nil)
		})

		Describe("Machinepools", Ordered, func() {
			It("Rejects spot flags on classic machine pools", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponse))
				t.SetCluster(clusterId, mockClassicClusterReady)
				args := NewEditMachinepoolUserOptions()
				args.machinepool = nodePoolId
				runner := EditMachinePoolRunner(args)
				cmd := NewEditMachinePoolCommand()
				Expect(cmd.Flag("cluster").Value.Set(clusterId)).To(Succeed())
				Expect(cmd.Flags().Set("use-spot-instances", "true")).To(Succeed())
				err := runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(MatchError(
					"spot instance configuration is only supported for Hosted Control Plane machine pools"))
			})
			It("Able to edit machinepool with no issues", func() {
				// First get
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponse))
				// Edit
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
				t.SetCluster(clusterId, mockClassicClusterReady)
				args := NewEditMachinepoolUserOptions()
				args.machinepool = nodePoolId
				runner := EditMachinePoolRunner(args)
				cmd := NewEditMachinePoolCommand()
				Expect(cmd.Flag("cluster").Value.Set(clusterId)).To(Succeed())
				Expect(cmd.Flags().Set("labels", "test=test")).To(Succeed())
				Expect(cmd.Flags().Set("replicas", "1")).To(Succeed())
				Expect(runner(context.Background(), t.RosaRuntime, cmd, []string{})).To(Succeed())
			})
			It("Machinepool ID passed in without flag in random location", func() {
				// First get
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponse))
				// Edit
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
				t.SetCluster(clusterId, mockClassicClusterReady)
				args := NewEditMachinepoolUserOptions()
				args.machinepool = nodePoolId
				runner := EditMachinePoolRunner(args)
				cmd := NewEditMachinePoolCommand()
				Expect(cmd.Flag("cluster").Value.Set(clusterId)).To(Succeed())
				Expect(cmd.Flags().Set("labels", "test=test")).To(Succeed())
				Expect(cmd.Flags().Set("min-replicas", "1")).To(Succeed())
				Expect(cmd.Flags().Set("max-replicas", "1")).To(Succeed())
				Expect(cmd.Flags().Set("enable-autoscaling", "true")).To(Succeed())
				Expect(runner(context.Background(), t.RosaRuntime, cmd,
					[]string{})).To(Succeed())
			})
		})

		Describe("Nodepools", func() {
			It("exposes the spot edit flags", func() {
				cmd := NewEditMachinePoolCommand()
				Expect(cmd.Flags().Lookup("use-spot-instances")).ToNot(BeNil())
				Expect(cmd.Flags().Lookup("spot-max-price")).ToNot(BeNil())
			})
			It("Able to edit nodepool with no issues", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				// First get
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				// Second get
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				// Edit
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
				t.SetCluster(clusterId, mockClusterReady)
				args := NewEditMachinepoolUserOptions()
				args.machinepool = nodePoolId
				runner := EditMachinePoolRunner(args)
				cmd := NewEditMachinePoolCommand()
				Expect(cmd.Flags().Set("enable-autoscaling", "true")).To(Succeed())
				Expect(cmd.Flags().Set("labels", "test=test")).To(Succeed())
				Expect(cmd.Flag("cluster").Value.Set(clusterId)).To(Succeed())
				Expect(cmd.Flags().Set("min-replicas", "2")).To(Succeed())
				Expect(cmd.Flags().Set("max-replicas", "10")).To(Succeed())
				Expect(cmd.Flag("yes").Value.Set("true")).To(Succeed())
				Expect(runner(context.Background(), t.RosaRuntime, cmd,
					[]string{"--machinepool", nodePoolId})).To(Succeed())
			})
			It("No need for --machinepool (ID by itself)", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				// First get
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolAutoResponse))
				// Second get
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				// Edit
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
				t.SetCluster(clusterId, mockClusterReady)
				args := NewEditMachinepoolUserOptions()
				args.machinepool = nodePoolId
				args.autoscalingEnabled = true
				args.maxReplicas = 10
				args.minReplicas = 2
				runner := EditMachinePoolRunner(args)
				cmd := NewEditMachinePoolCommand()
				Expect(cmd.Flag("cluster").Value.Set(clusterId)).To(Succeed())
				Expect(cmd.Flag("yes").Value.Set("true")).To(Succeed())
				Expect(cmd.Flags().Set("labels", "test=test")).To(Succeed())
				Expect(cmd.Flags().Set("enable-autoscaling", "true")).To(Succeed())
				Expect(cmd.Flags().Set("min-replicas", "2")).To(Succeed())
				Expect(cmd.Flags().Set("max-replicas", "10")).To(Succeed())
				Expect(runner(context.Background(), t.RosaRuntime, cmd, []string{nodePoolId, "--min-replicas",
					"2", "--enable-autoscaling", "true", "--interactive", "false", "--max-replicas",
					"10"})).To(Succeed())
			})
			It("Rejects modifying spot on an existing nodepool with guidance to recreate", func() {
				spotNodePool, err := cmv1.NewNodePool().ID(nodePoolId).Version(version).
					AWSNodePool(cmv1.NewAWSNodePool().InstanceType("m5.xlarge")).
					AvailabilityZone("us-east-1a").Build()
				Expect(err).ToNot(HaveOccurred())
				spotNodePoolResponse := formatNodePoolResource(spotNodePool)

				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, spotNodePoolResponse))
				t.SetCluster(clusterId, mockClusterReady)
				args := NewEditMachinepoolUserOptions()
				args.machinepool = nodePoolId
				runner := EditMachinePoolRunner(args)
				cmd := NewEditMachinePoolCommand()
				Expect(cmd.Flag("cluster").Value.Set(clusterId)).To(Succeed())
				Expect(cmd.Flag("yes").Value.Set("true")).To(Succeed())
				Expect(cmd.Flags().Set("use-spot-instances", "true")).To(Succeed())
				Expect(cmd.Flags().Set("spot-max-price", "1.00")).To(Succeed())

				err = runner(context.Background(), t.RosaRuntime, cmd, []string{nodePoolId})
				Expect(err).To(MatchError(ContainSubstring(
					"delete the machine pool and create a new one with the desired spot settings")))
			})
			It("Rejects disabling spot instances on nodepool edits for now", func() {
				existingSpotNodePool, err := cmv1.NewNodePool().ID(nodePoolId).Version(version).
					AWSNodePool(cmv1.NewAWSNodePool().InstanceType("m5.xlarge").
						SpotMarketOptions(cmv1.NewAwsNodePoolSpotMarketOptions().MaxPrice("1.00"))).
					AvailabilityZone("us-east-1a").Build()
				Expect(err).ToNot(HaveOccurred())
				existingSpotNodePoolResponse := formatNodePoolResource(existingSpotNodePool)

				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, existingSpotNodePoolResponse))
				t.SetCluster(clusterId, mockClusterReady)
				args := NewEditMachinepoolUserOptions()
				args.machinepool = nodePoolId
				runner := EditMachinePoolRunner(args)
				cmd := NewEditMachinePoolCommand()
				Expect(cmd.Flag("cluster").Value.Set(clusterId)).To(Succeed())
				Expect(cmd.Flag("yes").Value.Set("true")).To(Succeed())
				Expect(cmd.Flags().Set("use-spot-instances", "false")).To(Succeed())

				err = runner(context.Background(), t.RosaRuntime, cmd, []string{nodePoolId})
				Expect(err).To(MatchError("disabling spot instances on hosted machine pools is not supported yet"))
			})
		})
	})
})

func formatNodePoolResource(nodePool *cmv1.NodePool) string {
	var output bytes.Buffer
	Expect(cmv1.MarshalNodePool(nodePool, &output)).To(Succeed())
	return output.String()
}
