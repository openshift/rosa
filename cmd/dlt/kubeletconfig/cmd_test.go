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

var _ = Describe("delete kubeletconfig", func() {

	It("Correctly builds the command", func() {
		cmd := NewDeleteKubeletConfigCommand()
		Expect(cmd).NotTo(BeNil())

		Expect(cmd.Use).To(Equal(use))
		Expect(cmd.Short).To(Equal(short))
		Expect(cmd.Long).To(Equal(long))
		Expect(cmd.Args).NotTo(BeNil())
		Expect(cmd.Run).NotTo(BeNil())

		Expect(cmd.Flags().Lookup("cluster")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup(PodPidsLimitOption)).To(BeNil())
		Expect(cmd.Flags().Lookup(NameOption)).NotTo(BeNil())
	})

	Context("Delete KubeletConfig Runner", func() {

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

			runner := DeleteKubeletConfigRunner(NewKubeletConfigOptions())
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("There is no cluster with identifier or name 'cluster'"))
		})

		It("Deletes KubeletConfig by name for HCP Clusters", func() {

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
			t.ApiServer.RouteToHandler(http.MethodDelete,
				fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/kubelet_configs/%s", cluster.ID(), kubeletConfig.ID()),
				RespondWithJSON(http.StatusOK, FormatResource(kubeletConfig)))
			t.SetCluster("cluster", cluster)

			options := NewKubeletConfigOptions()
			options.Name = "testing"

			runner := DeleteKubeletConfigRunner(options)
			t.StdOutReader.Record()

			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal("INFO: Successfully deleted KubeletConfig for cluster 'cluster'\n"))
		})

		It("Fails to delete KubeletConfig by name for HCP Clusters if the KubeletConfig does not exist", func() {
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
			t.SetCluster("cluster", cluster)

			options := NewKubeletConfigOptions()
			options.Name = "testing"

			runner := DeleteKubeletConfigRunner(options)

			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Failed to delete KubeletConfig for cluster 'cluster'"))
		})
	})
})
