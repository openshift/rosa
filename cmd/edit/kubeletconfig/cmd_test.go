package kubeletconfig

import (
	"context"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	. "github.com/openshift/rosa/pkg/kubeletconfig"
	"github.com/openshift/rosa/pkg/output"
	. "github.com/openshift/rosa/pkg/test"
)

var _ = Describe("edit kubeletconfig", func() {

	It("Correctly builds the command", func() {
		cmd := NewEditKubeletConfigCommand()
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

	Context("Edit KubeletConfig Runner", func() {

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

			runner := EditKubeletConfigRunner(NewKubeletConfigOptions())
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

			runner := EditKubeletConfigRunner(NewKubeletConfigOptions())
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("Cluster 'cluster' is not yet ready. Current state is 'installing'"))

		})

		It("Returns an error if no kubeletconfig exists for classic cluster", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			config := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
				k.Name("test").PodPidsLimit(10000).ID("foo")
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusNotFound, FormatResource(config)))
			t.SetCluster("cluster", nil)

			runner := EditKubeletConfigRunner(NewKubeletConfigOptions())
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("The specified KubeletConfig does not exist for cluster 'cluster'. " +
					"You should first create it via 'rosa create kubeletconfig'"))
		})

		It("Returns an error if it fails to read the kubeletconfig from OCM for classic cluster", func() {

			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusInternalServerError, "{}"))
			t.SetCluster("cluster", nil)

			runner := EditKubeletConfigRunner(NewKubeletConfigOptions())
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("Failed to fetch KubeletConfig configuration for cluster "))
		})

		It("Returns an error if no kubeletconfig exists for HCP cluster", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				b := cmv1.HypershiftBuilder{}
				b.Enabled(true)
				c.Hypershift(&b)
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, FormatKubeletConfigList([]*cmv1.KubeletConfig{})))
			t.SetCluster("cluster", nil)

			options := NewKubeletConfigOptions()
			options.Name = "testing"
			options.PodPidsLimit = 10000

			runner := EditKubeletConfigRunner(options)
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("The specified KubeletConfig does not exist for cluster 'cluster'. " +
					"You should first create it via 'rosa create kubeletconfig'"))
		})

		It("Updates KubeletConfig for HCP Clusters", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				b := cmv1.HypershiftBuilder{}
				b.Enabled(true)
				c.Hypershift(&b)
			})

			kubeletConfig := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
				k.ID("testing").PodPidsLimit(5000).Name("testing")
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, FormatKubeletConfigList([]*cmv1.KubeletConfig{kubeletConfig})))
			t.ApiServer.RouteToHandler(http.MethodPatch,
				fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/kubelet_configs/%s", cluster.ID(), kubeletConfig.ID()),
				RespondWithJSON(http.StatusOK, FormatResource(kubeletConfig)))

			t.SetCluster("cluster", nil)

			options := NewKubeletConfigOptions()
			options.Name = "testing"
			options.PodPidsLimit = 10000

			runner := EditKubeletConfigRunner(options)
			t.StdOutReader.Record()

			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal("INFO: Successfully updated KubeletConfig for cluster 'cluster'\n"))
		})
	})
})
