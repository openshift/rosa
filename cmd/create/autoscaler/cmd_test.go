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

var _ = Describe("create autoscaler", func() {

	It("Correctly builds the command", func() {
		cmd := NewCreateAutoscalerCommand()
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

	Context("Create Autoscaler Runner", func() {

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

			runner := CreateAutoscalerRunner(&clusterautoscaler.AutoscalerArgs{})
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("There is no cluster with identifier or name 'cluster'"))
		})

		It("Returns an error if the cluster is not classic cluster", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				b := cmv1.HypershiftBuilder{}
				b.Enabled(true)
				c.Hypershift(&b)
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.SetCluster("cluster", nil)

			runner := CreateAutoscalerRunner(&clusterautoscaler.AutoscalerArgs{})
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("Hosted Control Plane clusters do not support cluster-autoscaler configuration"))
		})

		It("Returns an error if the cluster is not ready", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateInstalling)
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.SetCluster("cluster", nil)

			runner := CreateAutoscalerRunner(&clusterautoscaler.AutoscalerArgs{})
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("Cluster 'cluster' is not yet ready. Current state is 'installing'"))
		})

		It("Returns an error if an autoscaler exists for classic cluster", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})
			autoscaler := test.MockAutoscaler(func(a *cmv1.ClusterAutoscalerBuilder) {
				a.MaxNodeProvisionTime("10m")
				a.BalancingIgnoredLabels("foo", "bar")
				a.PodPriorityThreshold(10)
				a.LogVerbosity(2)
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, FormatResource(autoscaler)))
			t.SetCluster("cluster", nil)

			runner := CreateAutoscalerRunner(&clusterautoscaler.AutoscalerArgs{})
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal(fmt.Sprintf("Autoscaler for cluster '%s' already exists. "+
					"You should edit it via 'rosa edit autoscaler'", "cluster")))
		})

		It("Creates Austoscaler", func() {
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
				RespondWithJSON(http.StatusNotFound, "{}"))
			t.ApiServer.RouteToHandler(http.MethodPost,
				fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/autoscaler", cluster.ID()),
				CombineHandlers(
					RespondWithJSON(http.StatusOK, FormatResource(autoscaler)),
					VerifyJQ(`.log_verbosity`, 3.0),
					VerifyJQ(`.resource_limits.max_nodes_total`, 20.0),
					VerifyJQ(`.resource_limits.cores.max`, 0.0),
				))
			args := &clusterautoscaler.AutoscalerArgs{}
			args.LogVerbosity = 3
			args.ResourceLimits.MaxNodesTotal = 20
			runner := CreateAutoscalerRunner(args)
			cmd := NewCreateAutoscalerCommand()
			cmd.Flags().Set("log-verbosity", "3")
			cmd.Flags().Set("max-nodes-total", "20")
			t.SetCluster("cluster", nil)
			err := runner(context.Background(), t.RosaRuntime, cmd, nil)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
