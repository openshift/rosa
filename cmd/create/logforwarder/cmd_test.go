package logforwarder

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/logforwarding"
	"github.com/openshift/rosa/pkg/output"
	. "github.com/openshift/rosa/pkg/test"
)

var _ = Describe("create LogForwarder", func() {

	It("Correctly builds the command", func() {
		cmd := NewCreateLogForwarderCommand()
		Expect(cmd).NotTo(BeNil())

		Expect(cmd.Use).To(Equal(use))
		Expect(cmd.Short).To(Equal(short))
		Expect(cmd.Long).To(Equal(long))
		Expect(cmd.Run).NotTo(BeNil())

		Expect(cmd.Flags().Lookup("cluster")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("interactive")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup(logforwarding.FlagName)).NotTo(BeNil())
	})

	Context("Create LogForwarder Runner", func() {

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

			userOptions := NewCreateLogForwarderUserOptions()
			userOptions.logFwdConfig = ""

			runner := CreateLogForwarderRunner(userOptions)
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("There is no cluster with identifier or name 'cluster'"))
		})

		It("Returns an error if the cluster is not an HCP cluster", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.SetCluster("cluster", cluster)

			userOptions := NewCreateLogForwarderUserOptions()
			userOptions.logFwdConfig = ""

			runner := CreateLogForwarderRunner(userOptions)
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("log forwarders are only supported for Hosted Control Plane clusters"))
		})

		It("Returns an error if the cluster is a not-ready HCP cluster", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				b := cmv1.HypershiftBuilder{}
				b.Enabled(true)
				c.Hypershift(&b)
				c.State(cmv1.ClusterStateInstalling)
			})

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.SetCluster("cluster", cluster)

			userOptions := NewCreateLogForwarderUserOptions()
			userOptions.logFwdConfig = ""

			runner := CreateLogForwarderRunner(userOptions)
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cluster 'cluster' is not yet ready"))
		})
	})
})
