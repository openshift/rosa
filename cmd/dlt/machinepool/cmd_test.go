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
	nodePoolName = "nodepool85"
	clusterId    = "24vf9iitg3p6tlml88iml6j6mu095mh8"
)

var _ = Describe("Delete machine pool", func() {
	Context("Delete machine pool command", func() {
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
		mpResponse := formatMachinePool()

		var t *test.TestingRuntime

		BeforeEach(func() {
			t = test.NewTestRuntime()
			SetOutput("")
		})
		It("Fails if we are not specifying a machine pool name", func() {
			runner := DeleteMachinePoolRunner(NewDeleteMachinepoolUserOptions())
			err := runner(context.Background(), t.RosaRuntime, NewDeleteMachinePoolCommand(), []string{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("You need to specify a machine pool name"))
		})
		Context("Hypershift", func() {
			It("Works without passing `--machinepool`", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				// First get
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				// Delete
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
				args := NewDeleteMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DeleteMachinePoolRunner(args)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDeleteMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = cmd.Flag("yes").Value.Set("true")
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{"--machinepool", nodePoolName, "-y"})
				Expect(err).To(BeNil())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(fmt.Sprintf("INFO: Successfully deleted machine pool '%s' from "+
					"hosted cluster '%s'\n", nodePoolName, clusterId)))
			})
			It("Pass a machine pool name through argv but it is not found", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
				args := NewDeleteMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DeleteMachinePoolRunner(args)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDeleteMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{"--machinepool", nodePoolName, "-y"})
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Error deleting machinepool: machine pool "+
					"'%s' does not exist for hosted cluster '%s'", nodePoolName, clusterId)))
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(""))
			})
			It("Pass only machine pool name, gives prompt successfully", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				// First get
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				args := NewDeleteMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DeleteMachinePoolRunner(args)
				cmd := NewDeleteMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{nodePoolName})
				Expect(err).To(BeNil())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(ContainSubstring(fmt.Sprintf("Are you sure you want to delete machine pool "+
					"'%s' on hosted cluster '%s'?", nodePoolName, clusterId)))
			})
		})
		Context("ROSA Classic", func() {
			It("Works without passing `--machinepool`", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				// First get
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponse))
				// Delete
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
				args := NewDeleteMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DeleteMachinePoolRunner(args)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDeleteMachinePoolCommand()
				err = cmd.Flag("yes").Value.Set("true")
				Expect(err).ToNot(HaveOccurred())
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{nodePoolName})
				Expect(err).To(BeNil())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(fmt.Sprintf("INFO: Successfully deleted machine pool '%s' from "+
					"cluster '%s'\n", nodePoolName, clusterId)))
			})
			It("Pass a machine pool name through argv but it is not found", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
				args := NewDeleteMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DeleteMachinePoolRunner(args)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDeleteMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{"--machinepool", nodePoolName, "-y"})
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Error deleting machinepool: failed to "+
					"get machine pool '%s' for cluster '%s'", nodePoolName,
					clusterId)))
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(""))
			})
			It("Pass only machine pool name, gives prompt successfully", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				// First get
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, mpResponse))
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				args := NewDeleteMachinepoolUserOptions()
				args.machinepool = nodePoolName
				runner := DeleteMachinePoolRunner(args)
				cmd := NewDeleteMachinePoolCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd,
					[]string{nodePoolName})
				Expect(err).To(BeNil())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(ContainSubstring(fmt.Sprintf("Are you sure you want to delete machine pool "+
					"'%s' on cluster '%s'?", nodePoolName, clusterId)))
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

// formatMachinePool simulates the output of APIs for a fake machine pool list
func formatMachinePool() string {
	awsMachinePoolPool := cmv1.NewAWSMachinePool().SpotMarketOptions(cmv1.NewAWSSpotMarketOptions().MaxPrice(5))
	machinePool, err := cmv1.NewMachinePool().ID(nodePoolName).AWS(awsMachinePoolPool).InstanceType("m5.xlarge").
		AvailabilityZones("us-east-1a", "us-east-1b", "us-east-1c").Build()
	Expect(err).ToNot(HaveOccurred())
	return fmt.Sprintf("{\n  \"items\": [\n    %s\n  ],\n  \"page\": 0,\n  \"size\": 1,\n  \"total\": 1\n}",
		test.FormatResource(machinePool))
}
