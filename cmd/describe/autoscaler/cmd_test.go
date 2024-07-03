package autoscaler

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/clusterautoscaler"
	"github.com/openshift/rosa/pkg/output"
	. "github.com/openshift/rosa/pkg/test"
)

func TestDescribeAutoscaler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rosa describe autoscaler")
}

var _ = Describe("rosa describe autoscaler", func() {
	Context("Create Command", func() {
		It("Returns Command", func() {

			cmd := NewDescribeAutoscalerCommand()
			Expect(cmd).NotTo(BeNil())

			Expect(cmd.Use).To(Equal(use))
			Expect(cmd.Example).To(Equal(example))
			Expect(cmd.Short).To(Equal(short))
			Expect(cmd.Long).To(Equal(long))
			Expect(cmd.Args).NotTo(BeNil())
			Expect(cmd.Run).NotTo(BeNil())

			flag := cmd.Flags().Lookup("cluster")
			Expect(flag).NotTo(BeNil())

			flag = cmd.Flags().Lookup("output")
			Expect(flag).NotTo(BeNil())
		})
	})

	Context("Execute command", func() {

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

			runner := DescribeAutoscalerRunner()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("There is no cluster with identifier or name 'cluster'"))
		})

		It("Returns an error if the cluster is HCP", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				h := &cmv1.HypershiftBuilder{}
				h.Enabled(true)
				c.Hypershift(h)
			})

			t.SetCluster(cluster.Name(), cluster)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))

			runner := DescribeAutoscalerRunner()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring(clusterautoscaler.NoHCPAutoscalerSupportMessage))
		})

		It("Returns an error if the cluster is not ready", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateInstalling)
			})

			t.SetCluster(cluster.Name(), cluster)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))

			runner := DescribeAutoscalerRunner()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring(" is not yet ready"))
		})

		It("Returns an error if OCM API fails to return autoscaler", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.SetCluster(cluster.Name(), cluster)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.RouteToHandler(
				"GET",
				fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/autoscaler", cluster.ID()),
				RespondWithJSON(http.StatusInternalServerError, "{}"))

			runner := DescribeAutoscalerRunner()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("status is 500"))

		})

		It("Returns an error if no autoscaler exists", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.SetCluster(cluster.Name(), cluster)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, "{}"))

			runner := DescribeAutoscalerRunner()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("No autoscaler exists for cluster 'cluster'"))

		})

		It("Prints the autoscaler to stdout", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.SetCluster(cluster.Name(), cluster)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))

			autoscaler := MockAutoscaler(nil)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResource(autoscaler)))

			runner := DescribeAutoscalerRunner()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).NotTo(HaveOccurred())
		})

		It("Prints the autoscaler in JSON", func() {
			output.SetOutput("json")
			Expect(output.HasFlag()).To(BeTrue())

			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.SetCluster(cluster.Name(), cluster)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))

			autoscaler := MockAutoscaler(nil)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResource(autoscaler)))

			runner := DescribeAutoscalerRunner()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).NotTo(HaveOccurred())

		})

		It("Prints the autoscaler in YAML", func() {
			output.SetOutput("yaml")
			Expect(output.HasFlag()).To(BeTrue())

			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.SetCluster(cluster.Name(), cluster)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))

			autoscaler := MockAutoscaler(nil)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResource(autoscaler)))

			runner := DescribeAutoscalerRunner()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).NotTo(HaveOccurred())
		})
	})
})
