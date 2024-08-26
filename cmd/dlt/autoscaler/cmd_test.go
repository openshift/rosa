package autoscaler

import (
	"context"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/output"
	. "github.com/openshift/rosa/pkg/test"
)

var _ = Describe("delete autoscaler", func() {

	It("Correctly builds the command", func() {
		cmd := NewDeleteAutoscalerCommand()
		Expect(cmd).NotTo(BeNil())

		Expect(cmd.Use).To(Equal(use))
		Expect(cmd.Short).To(Equal(short))
		Expect(cmd.Long).To(Equal(long))
		Expect(cmd.Run).NotTo(BeNil())

		Expect(cmd.Flags().Lookup("yes")).NotTo(BeNil())
	})

	Context("Delete Autoscaler Runner", func() {

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

			runner := DeleteAutoscalerRunner()
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

			runner := DeleteAutoscalerRunner()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("Hosted Control Plane clusters do not support cluster-autoscaler configuration"))
		})

		It("Deletes Austoscaler", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.RouteToHandler(http.MethodGet,
				fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/autoscaler", cluster.ID()),
				RespondWithJSON(http.StatusOK, ""))
			runner := DeleteAutoscalerRunner()
			t.SetCluster("cluster", nil)
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
