package kubeletconfig

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	. "github.com/openshift/rosa/pkg/test"
)

var classicOutputWithName = `
ID:                                   bar
Name:                                 foo
Pod Pids Limit:                       10000
`

var classicOutPutNoName = `
ID:                                   bar
Name:                                 -
Pod Pids Limit:                       10000
`

var hcpOutputWithName = `
ID:                                   bar
Name:                                 foo
Pod Pids Limit:                       10000
MachinePools Using This KubeletConfig:
 - testing
`

var hcpOutputNoName = `
ID:                                   bar
Name:                                 -
Pod Pids Limit:                       10000
MachinePools Using This KubeletConfig:
 - testing
`

var _ = Describe("KubeletConfig Output", func() {
	It("Correctly Prints KubeletConfigList for Tabuluar Output", func() {

		kubeletConfig := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
			k.Name("test").PodPidsLimit(10000).ID("foo")
		})

		kubeletConfig2 := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
			k.Name("").PodPidsLimit(20000).ID("bar")
		})

		output := PrintKubeletConfigsForTabularOutput([]*cmv1.KubeletConfig{kubeletConfig, kubeletConfig2})
		Expect(output).To(Equal("ID\tNAME\tPOD PIDS LIMIT\nfoo\ttest\t10000\nbar\t-\t20000\n"))
	})

	It("Prints KubeletConfig For Classic", func() {
		kubeletConfig := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
			k.Name("foo").ID("bar").PodPidsLimit(10000)
		})

		output := PrintKubeletConfigForClassic(kubeletConfig)
		Expect(output).To(Equal(classicOutputWithName))
	})

	It("Prints KubeletConfig For Classic with no Name", func() {
		kubeletConfig := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
			k.ID("bar").PodPidsLimit(10000)
		})

		output := PrintKubeletConfigForClassic(kubeletConfig)
		Expect(output).To(Equal(classicOutPutNoName))
	})

	It("Prints KubeletConfig for HCP", func() {
		kubeletConfig := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
			k.Name("foo").ID("bar").PodPidsLimit(10000)
		})

		nodePool := MockNodePool(func(n *cmv1.NodePoolBuilder) {
			n.ID("testing")
		})

		output := PrintKubeletConfigForHcp(kubeletConfig, []*cmv1.NodePool{nodePool})
		Expect(output).To(Equal(hcpOutputWithName))
	})

	It("Prints KubeletConfig for HCP with no name", func() {
		kubeletConfig := MockKubeletConfig(func(k *cmv1.KubeletConfigBuilder) {
			k.ID("bar").PodPidsLimit(10000)
		})

		nodePool := MockNodePool(func(n *cmv1.NodePoolBuilder) {
			n.ID("testing")
		})

		output := PrintKubeletConfigForHcp(kubeletConfig, []*cmv1.NodePool{nodePool})
		Expect(output).To(Equal(hcpOutputNoName))
	})
})
