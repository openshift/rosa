package autoscaler

import (
	"context"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/ghttp"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/clusterautoscaler"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/test"
	. "github.com/openshift/rosa/pkg/test"
)

var _ = Describe("edit autoscaler", func() {

	It("Correctly builds the command", func() {
		cmd := NewEditAutoscalerCommand()
		Expect(cmd).NotTo(BeNil())

		Expect(cmd.Use).To(Equal(use))
		Expect(cmd.Short).To(Equal(short))
		Expect(cmd.Long).To(Equal(long))
		Expect(cmd.Run).NotTo(BeNil())

		Expect(cmd.Flags().Lookup("balance-similar-node-groups")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("log-verbosity")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("max-nodes-total")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("scale-down-enabled")).NotTo(BeNil())
	})

	Context("Edit Autoscaler Runner", func() {

		var t *TestingRuntime

		BeforeEach(func() {
			t = NewTestRuntime()
			output.SetOutput("")
		})

		AfterEach(func() {
			output.SetOutput("")
		})

		It("Returns an error if the cluster does not exist", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList(make([]*cmv1.Cluster, 0))))
			t.SetCluster("cluster", nil)

			runner := EditAutoscalerRunner(&clusterautoscaler.AutoscalerArgs{})
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("There is no cluster with identifier or name 'cluster'"))
		})

		It("Returns an error if the cluster is not ready", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateInstalling)
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.SetCluster("cluster", nil)

			runner := EditAutoscalerRunner(&clusterautoscaler.AutoscalerArgs{})
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("Cluster 'cluster' is not yet ready. Current state is 'installing'"))
		})

		It("Returns an error if no autoscaler exists for classic cluster", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusNotFound, "{}"))
			t.SetCluster("cluster", nil)

			runner := EditAutoscalerRunner(&clusterautoscaler.AutoscalerArgs{})
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal(fmt.Sprintf("No autoscaler for cluster '%s' has been found. "+
					"You should first create it via 'rosa create autoscaler'", "cluster")))
		})

		It("Updates Austoscaler", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			autoscaler := test.MockAutoscaler(func(a *cmv1.ClusterAutoscalerBuilder) {
				a.MaxNodeProvisionTime("10m")
				a.BalancingIgnoredLabels("foo", "bar")
				a.PodPriorityThreshold(10)
				a.LogVerbosity(2)
				a.MaxPodGracePeriod(10)
				a.IgnoreDaemonsetsUtilization(true)
				a.SkipNodesWithLocalStorage(true)
				a.BalanceSimilarNodeGroups(false)

				sd := &cmv1.AutoscalerScaleDownConfigBuilder{}
				sd.Enabled(true)
				sd.DelayAfterFailure("10m")
				sd.DelayAfterAdd("5m")
				sd.DelayAfterDelete("20m")
				sd.UnneededTime("25m")
				sd.UtilizationThreshold("0.5")
				a.ScaleDown(sd)

				rl := &cmv1.AutoscalerResourceLimitsBuilder{}
				rl.MaxNodesTotal(10)

				mem := &cmv1.ResourceRangeBuilder{}
				mem.Max(10).Min(5)
				rl.Memory(mem)

				cores := &cmv1.ResourceRangeBuilder{}
				cores.Min(20).Max(30)
				rl.Cores(cores)

				gpus := &cmv1.AutoscalerResourceLimitsGPULimitBuilder{}
				gpus.Type("nvidia.com/gpu")

				gpuRR := &cmv1.ResourceRangeBuilder{}
				gpuRR.Max(20).Min(10)
				gpus.Range(gpuRR)

				rl.GPUS(gpus)
				a.ResourceLimits(rl)
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.RouteToHandler(http.MethodGet,
				fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/autoscaler", cluster.ID()),
				RespondWithJSON(http.StatusOK, FormatResource(autoscaler)))
			t.ApiServer.RouteToHandler(http.MethodPatch,
				fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/autoscaler", cluster.ID()),
				CombineHandlers(
					RespondWithJSON(http.StatusOK, FormatResource(autoscaler)),
					VerifyJQ(`.log_verbosity`, 1.0),
					VerifyJQ(`.resource_limits.max_nodes_total`, 20.0),
					VerifyJQ(`.resource_limits.cores.max`, 30.0),
				))
			args := &clusterautoscaler.AutoscalerArgs{}
			args.LogVerbosity = 1
			args.ResourceLimits.MaxNodesTotal = 20
			runner := EditAutoscalerRunner(args)
			cmd := NewEditAutoscalerCommand()
			cmd.Flags().Set("log-verbosity", "1")
			cmd.Flags().Set("max-nodes-total", "20")
			t.SetCluster("cluster", nil)
			err := runner(context.Background(), t.RosaRuntime, cmd, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Unsupported flags are blocked when using Hosted CP cluster for autoscaler", func() {
			cmd := NewEditAutoscalerCommand()

			cmd.Flags().Set("balance-similar-node-groups", "true")
			cmd.Flag("balance-similar-node-groups").Changed = true

			ok, err := clusterautoscaler.ValidateAutoscalerFlagsForHostedCp("", cmd)
			Expect(ok).To(BeFalse())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf(clusterautoscaler.HcpError, "balance-similar-node-groups",
				"max-nodes-total", "max-pod-grace-period", "max-node-provision-time",
				"pod-priority-threshold")))

			cmd = NewEditAutoscalerCommand()

			cmd.Flags().Set("max-cores", "true")
			cmd.Flag("max-cores").Changed = true

			ok, err = clusterautoscaler.ValidateAutoscalerFlagsForHostedCp("", cmd)
			Expect(ok).To(BeFalse())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf(clusterautoscaler.HcpError, "max-cores",
				"max-nodes-total", "max-pod-grace-period", "max-node-provision-time",
				"pod-priority-threshold")))
		})

		It("Supported flags work for Hosted CP cluster autoscaler", func() {
			cmd := NewEditAutoscalerCommand()

			cmd.Flags().Set("max-nodes-total", "10")
			cmd.Flag("max-nodes-total").Changed = true
			cmd.Flags().Set("max-pod-grace-period", "800")
			cmd.Flag("max-pod-grace-period").Changed = true
			cmd.Flags().Set("max-node-provision-time", "10s")
			cmd.Flag("max-node-provision-time").Changed = true
			cmd.Flags().Set("pod-priority-threshold", "-8")
			cmd.Flag("pod-priority-threshold").Changed = true

			ok, err := clusterautoscaler.ValidateAutoscalerFlagsForHostedCp("", cmd)
			Expect(ok).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
