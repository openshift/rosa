package tuningconfigs

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TuningConfigs Create Tests", func() {
	Context("buildTuningConfigFromInputFile", func() {
		name := "test-tuning-config"
		clusterKey := "test-cluster"

		It("OK: Should work for json format", func() {
			path := "spec.json"
			tuningConfig, err := buildTuningConfigFromInputFile(path, name, clusterKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(tuningConfig.Name()).To(Equal(name))
		})

		It("OK: Should work for yaml format", func() {
			path := "spec.yaml"
			tuningConfig, err := buildTuningConfigFromInputFile(path, name, clusterKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(tuningConfig.Name()).To(Equal(name))
		})
	})
})
