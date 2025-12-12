package logforwarders

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

var _ = Describe("list logforwarders", func() {

	It("Correctly builds the command", func() {
		cmd := NewListLogForwardersCommand()
		Expect(cmd).NotTo(BeNil())

		Expect(cmd.Use).To(Equal(use))
		Expect(cmd.Short).To(Equal(short))
		Expect(cmd.Long).To(Equal(long))
		Expect(cmd.Aliases).To(ContainElements(aliases))
		Expect(cmd.Run).NotTo(BeNil())

		Expect(cmd.Flags().Lookup("cluster")).NotTo(BeNil())
	})

	Context("List LogForwarders Runner", func() {

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

			runner := ListLogForwardersRunner()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("There is no cluster with identifier or name 'cluster'"))
		})

		It("Returns an error if the cluster is not ready", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateInstalling)
			})

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.SetCluster("cluster", cluster)

			runner := ListLogForwardersRunner()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cluster 'cluster' is not yet ready"))
		})

		It("Returns an error if OCM API fails to list log forwarders", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.SetCluster("cluster", cluster)
			t.ApiServer.RouteToHandler(
				"GET",
				fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/control_plane/log_forwarders", cluster.ID()),
				RespondWithJSON(http.StatusInternalServerError, "{}"))

			runner := ListLogForwardersRunner()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get log forwarders for cluster 'cluster'"))
		})
	})
})
