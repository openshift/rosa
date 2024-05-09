package kubeletconfig

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

var tabularOutput = `ID   NAME      POD PIDS LIMIT
foo  testing   10000
bar  testing2  20000
`

var _ = Describe("List KubeletConfig Command", func() {

	Context("Create Command", func() {
		It("Creates the command correctly", func() {
			cmd := NewListKubeletConfigsCommand()
			Expect(cmd).NotTo(BeNil())

			Expect(cmd.Use).To(Equal(use))
			Expect(cmd.Short).To(Equal(short))
			Expect(cmd.Long).To(Equal(long))
			Expect(cmd.Aliases).To(ContainElements(alias))
			Expect(cmd.Args).NotTo(BeNil())
			Expect(cmd.Run).NotTo(BeNil())
			Expect(cmd.RunE).To(BeNil())

			flags := cmd.Flags()
			Expect(flags.Lookup("cluster")).NotTo(BeNil())
			Expect(flags.Lookup("output")).NotTo(BeNil())
		})
	})

	Context("Command Runner", func() {

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

			runner := ListKubeletConfigRunner()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("There is no cluster with identifier or name 'cluster'"))
		})

		It("Returns an error if OCM API fails to list KubeletConfigs", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.SetCluster(cluster.Name(), cluster)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.RouteToHandler(
				"GET",
				fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/kubelet_configs", cluster.ID()),
				RespondWithJSON(http.StatusInternalServerError, "{}"))

			runner := ListKubeletConfigRunner()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("status is 500"))
		})

		It("Prints message if there are no KubeletConfigs for the cluster", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.SetCluster(cluster.Name(), cluster)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatKubeletConfigList([]*cmv1.KubeletConfig{})))

			runner := ListKubeletConfigRunner()

			t.StdOutReader.Record()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal("INFO: There are no KubeletConfigs for cluster 'cluster'.\n"))
		})

		It("Prints empty json if there are no KubeletConfigs for the cluster", func() {
			output.SetOutput("json")
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.SetCluster(cluster.Name(), cluster)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatKubeletConfigList([]*cmv1.KubeletConfig{})))

			runner := ListKubeletConfigRunner()

			t.StdOutReader.Record()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal("[]\n"))
		})

		It("Prints empty yaml if there are no KubeletConfigs for the cluster", func() {
			output.SetOutput("yaml")
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.SetCluster(cluster.Name(), cluster)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatKubeletConfigList([]*cmv1.KubeletConfig{})))

			runner := ListKubeletConfigRunner()

			t.StdOutReader.Record()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal("[]\n"))
		})

		It("Prints tabular list of KubeletConfigs for the cluster", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			kubeletConfig := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
				k.Name("testing").ID("foo").PodPidsLimit(10000)
			})

			kubeletConfig2 := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
				k.Name("testing2").ID("bar").PodPidsLimit(20000)
			})

			t.SetCluster(cluster.Name(), cluster)
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatKubeletConfigList(
						[]*cmv1.KubeletConfig{kubeletConfig, kubeletConfig2})))

			runner := ListKubeletConfigRunner()

			t.StdOutReader.Record()
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal(tabularOutput))
		})
	})
})
