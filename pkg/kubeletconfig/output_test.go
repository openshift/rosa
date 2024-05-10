package kubeletconfig

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	. "github.com/openshift/rosa/pkg/test"
)

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
})
