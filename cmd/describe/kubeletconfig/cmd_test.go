package kubeletconfig

import (
	"context"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/interactive"
	. "github.com/openshift/rosa/pkg/kubeletconfig"
	"github.com/openshift/rosa/pkg/output"
	. "github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Describe KubeletConfig", func() {

	It("Correctly Builds the Command", func() {
		cmd := NewDescribeKubeletConfigCommand()
		Expect(cmd).NotTo(BeNil())

		Expect(cmd.Use).To(Equal(use))
		Expect(cmd.Short).To(Equal(short))
		Expect(cmd.Long).To(Equal(long))
		Expect(cmd.Args).NotTo(BeNil())
		Expect(cmd.Run).NotTo(BeNil())

		Expect(cmd.Flags().Lookup("cluster")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("interactive")).To(BeNil())
		Expect(cmd.Flags().Lookup(PodPidsLimitOption)).To(BeNil())
		Expect(cmd.Flags().Lookup(NameOption)).NotTo(BeNil())
	})

	Context("Describe KubeletConfig Runner", func() {

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
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatClusterList(make([]*cmv1.Cluster, 0))))
			t.SetCluster("cluster", nil)

			runner := DescribeKubeletConfigRunner(NewKubeletConfigOptions())
			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("There is no cluster with identifier or name 'cluster'"))
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
			t.SetCluster("cluster", cluster)

			runner := DescribeKubeletConfigRunner(NewKubeletConfigOptions())
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("The KubeletConfig specified does not exist for cluster 'cluster'"))
		})

		It("Returns an error if failing to read Classic KubeletConfig", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusInternalServerError, "{}"))
			t.SetCluster("cluster", cluster)

			runner := DescribeKubeletConfigRunner(NewKubeletConfigOptions())
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).To(HaveOccurred())
		})

		It("Returns an error if the KubeletConfig does not exist for HCP", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				b := cmv1.HypershiftBuilder{}
				b.Enabled(true)
				c.Hypershift(&b)
			})

			options := NewKubeletConfigOptions()
			options.Name = "testing"

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, FormatKubeletConfigList([]*cmv1.KubeletConfig{})))
			t.SetCluster("cluster", cluster)

			runner := DescribeKubeletConfigRunner(options)
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				Equal("The KubeletConfig specified does not exist for cluster 'cluster'"))
		})

		It("Returns an error if failing to read HCP KubeletConfig", func() {
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
				RespondWithJSON(http.StatusInternalServerError, "{}"))
			t.SetCluster("cluster", cluster)

			runner := DescribeKubeletConfigRunner(NewKubeletConfigOptions())
			err := runner(context.Background(), t.RosaRuntime, nil, nil)

			Expect(err).To(HaveOccurred())
		})

		It("Prints the KubeletConfig in JSON for HCP", func() {
			output.SetOutput("json")

			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				b := cmv1.HypershiftBuilder{}
				b.Enabled(true)
				c.Hypershift(&b)
			})

			config := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
				k.Name("foo").ID("bar").PodPidsLimit(10000)
			})

			options := NewKubeletConfigOptions()
			options.Name = "foo"

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, FormatKubeletConfigList([]*cmv1.KubeletConfig{config})))
			t.SetCluster("cluster", cluster)

			runner := DescribeKubeletConfigRunner(options)
			t.StdOutReader.Record()

			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal(
				"{\n  \"kind\": \"KubeletConfig\",\n  \"id\": \"bar\",\n  \"name\": \"foo\",\n  \"pod_pids_limit\": 10000\n}\n"))
		})

		It("Prints the KubeletConfig in YAML for HCP", func() {
			output.SetOutput("yaml")
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				b := cmv1.HypershiftBuilder{}
				b.Enabled(true)
				c.Hypershift(&b)
			})

			config := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
				k.Name("foo").ID("bar").PodPidsLimit(10000)
			})

			options := NewKubeletConfigOptions()
			options.Name = "foo"

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, FormatKubeletConfigList([]*cmv1.KubeletConfig{config})))
			t.SetCluster("cluster", cluster)

			runner := DescribeKubeletConfigRunner(options)
			t.StdOutReader.Record()

			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal("id: bar\nkind: KubeletConfig\nname: foo\npod_pids_limit: 10000\n"))
		})

		It("Prints the KubeletConfig for Classic", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			config := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
				k.Name("foo").ID("bar").PodPidsLimit(10000)
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, FormatResource(config)))
			t.SetCluster("cluster", cluster)

			runner := DescribeKubeletConfigRunner(NewKubeletConfigOptions())
			t.StdOutReader.Record()

			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal(PrintKubeletConfigForClassic(config)))
		})

		It("Prints the KubeletConfig for HCP", func() {
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				b := cmv1.HypershiftBuilder{}
				b.Enabled(true)
				c.Hypershift(&b)
			})

			config := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
				k.Name("foo").ID("bar").PodPidsLimit(10000)
			})

			nodePool := MockNodePool(func(n *cmv1.NodePoolBuilder) {
				n.ID("testing").KubeletConfigs(config.Name())
			})
			nodePools := []*cmv1.NodePool{nodePool}

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, FormatKubeletConfigList([]*cmv1.KubeletConfig{config})))
			t.ApiServer.RouteToHandler(http.MethodGet,
				fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/node_pools", cluster.ID()),
				RespondWithJSON(http.StatusOK, FormatNodePoolList(nodePools)))

			options := NewKubeletConfigOptions()
			options.Name = "foo"

			t.SetCluster("cluster", cluster)

			runner := DescribeKubeletConfigRunner(options)
			t.StdOutReader.Record()

			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal(PrintKubeletConfigForHcp(config, nodePools)))
		})

		It("Prints the KubeletConfig in JSON for Classic", func() {
			output.SetOutput("json")
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			config := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
				k.Name("foo").ID("bar").PodPidsLimit(10000)
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, FormatResource(config)))
			t.SetCluster("cluster", cluster)

			runner := DescribeKubeletConfigRunner(NewKubeletConfigOptions())
			t.StdOutReader.Record()

			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal(
				"{\n  \"kind\": \"KubeletConfig\",\n  \"id\": \"bar\",\n  \"name\": \"foo\",\n  \"pod_pids_limit\": 10000\n}\n"))
		})

		It("Prints the KubeletConfig in YAML for Classic", func() {
			output.SetOutput("yaml")
			cluster := MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
			})

			config := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
				k.Name("foo").ID("bar").PodPidsLimit(10000)
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, FormatClusterList([]*cmv1.Cluster{cluster})))
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, FormatResource(config)))
			t.SetCluster("cluster", cluster)

			runner := DescribeKubeletConfigRunner(NewKubeletConfigOptions())
			t.StdOutReader.Record()

			err := runner(context.Background(), t.RosaRuntime, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal("id: bar\nkind: KubeletConfig\nname: foo\npod_pids_limit: 10000\n"))
		})
	})
})
