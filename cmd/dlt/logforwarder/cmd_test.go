package logforwarder

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/output"
	. "github.com/openshift/rosa/pkg/test"
)

var _ = Describe("delete logforwarder", func() {

	It("Correctly builds the command", func() {
		cmd := NewDeleteLogForwarderCommand()
		Expect(cmd).NotTo(BeNil())

		Expect(cmd.Use).To(Equal(use))
		Expect(cmd.Short).To(Equal(short))
		Expect(cmd.Long).To(Equal(long))
		Expect(cmd.Run).NotTo(BeNil())

		Expect(cmd.Flags().Lookup("cluster")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("log-forwarder")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("yes")).NotTo(BeNil())
	})

	Context("Delete LogForwarder Runner", func() {

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

			userOptions := NewDeleteLogForwarderUserOptions()
			userOptions.logForwarder = "testid123"

			runner := DeleteLogForwarderRunner(userOptions)
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

			userOptions := NewDeleteLogForwarderUserOptions()
			userOptions.logForwarder = "testid123"

			runner := DeleteLogForwarderRunner(userOptions)
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cluster 'cluster' is not yet ready"))
		})

		It("Returns an error if log forwarder ID is not provided", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.SetCluster("cluster", cluster)

			userOptions := NewDeleteLogForwarderUserOptions()

			runner := DeleteLogForwarderRunner(userOptions)
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("you must specify a log forwarder ID with '--log-forwarder'"))
		})

		It("Returns an error if log forwarder ID has invalid format", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.SetCluster("cluster", cluster)

			userOptions := NewDeleteLogForwarderUserOptions()
			userOptions.logForwarder = "Invalid-ID"

			runner := DeleteLogForwarderRunner(userOptions)
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("log forwarder ID 'Invalid-ID' is not valid: it must " +
				"contain only lowercase letters and digits"))
		})
	})
})
