package kubeletconfig

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/interactive"
	. "github.com/openshift/rosa/pkg/kubeletconfig"
	"github.com/openshift/rosa/pkg/output"
	. "github.com/openshift/rosa/pkg/test"
)

var _ = Describe("create kubeletconfig", func() {

	It("Correctly builds the command", func() {
		cmd := NewCreateKubeletConfigCommand()
		Expect(cmd).NotTo(BeNil())

		Expect(cmd.Use).To(Equal(use))
		Expect(cmd.Short).To(Equal(short))
		Expect(cmd.Long).To(Equal(long))
		Expect(cmd.Args).NotTo(BeNil())
		Expect(cmd.Run).NotTo(BeNil())

		Expect(cmd.Flags().Lookup("cluster")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("interactive")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup(PodPidsLimitOption)).NotTo(BeNil())
		Expect(cmd.Flags().Lookup(NameOption)).NotTo(BeNil())
	})

	Context("CreateKubeletConfig Runner", func() {

		var t *TestingRuntime

		BeforeEach(func() {
			t = NewTestRuntime()
			output.SetOutput("")
			interactive.SetEnabled(false)
		})

		AfterEach(func() {
			output.SetOutput("")
			interactive.SetEnabled(false)
		})

		It("Returns an error if the cluster does not exist", func() {
			t.ApiServer.AppendHandlers(testing.RespondWithJSON(http.StatusOK, FormatClusterList(make([]*cmv1.Cluster, 0))))
			t.SetCluster("cluster", nil)

			runner := CreateKubeletConfigRunner(NewKubeletConfigOptions())
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
				testing.RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.SetCluster("cluster", nil)

			runner := CreateKubeletConfigRunner(NewKubeletConfigOptions())
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("Cluster 'cluster' is not yet ready. Current state is 'installing'"))

		})

		It("Returns an error if a kubeletconfig already exists for classic cluster", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			config := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
				k.Name("test").PodPidsLimit(10000).ID("foo")
			})

			t.ApiServer.AppendHandlers(
				testing.RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				testing.RespondWithJSON(http.StatusOK, FormatResource(config)))
			t.SetCluster("cluster", nil)

			runner := CreateKubeletConfigRunner(NewKubeletConfigOptions())
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("A KubeletConfig for cluster 'cluster' already exists." +
					" You should edit it via 'rosa edit kubeletconfig'"))
		})

		It("Returns an error if it fails to read the kubeletconfig from OCM for classic cluster", func() {

			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.ApiServer.AppendHandlers(
				testing.RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				testing.RespondWithJSON(http.StatusInternalServerError, "{}"))
			t.SetCluster("cluster", nil)

			runner := CreateKubeletConfigRunner(NewKubeletConfigOptions())
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("Failed getting KubeletConfig for cluster 'cluster'"))
		})

		It("Creates the KubeletConfig for HCP clusters", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				b := cmv1.HypershiftBuilder{}
				b.Enabled(true)
				c.Hypershift(&b)

			})

			kubeletConfig := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
				k.ID("test-id").PodPidsLimit(10000).Name("testing")
			})

			t.ApiServer.AppendHandlers(
				testing.RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				testing.RespondWithJSON(http.StatusCreated, FormatResource(kubeletConfig)))
			t.SetCluster("cluster", nil)

			options := NewKubeletConfigOptions()
			options.PodPidsLimit = 10000
			options.Name = "test"

			runner := CreateKubeletConfigRunner(options)
			t.StdOutReader.Record()

			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal("INFO: Successfully created KubeletConfig for cluster 'cluster'\n"))
		})

		It("Returns an error if failing to create the KubeletConfig for HCP Clusters", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				b := cmv1.HypershiftBuilder{}
				b.Enabled(true)
				c.Hypershift(&b)

			})

			kubeletConfig := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
				k.ID("test-id").PodPidsLimit(10000).Name("testing")
			})

			t.ApiServer.AppendHandlers(
				testing.RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				testing.RespondWithJSON(http.StatusBadRequest, FormatResource(kubeletConfig)))
			t.SetCluster("cluster", nil)

			options := NewKubeletConfigOptions()
			options.PodPidsLimit = 10000
			options.Name = "test"

			runner := CreateKubeletConfigRunner(options)

			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Failed creating KubeletConfig for cluster 'cluster':"))
		})
	})
})
