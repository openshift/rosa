package cluster

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validates OCP version", func() {

	const (
		nightly = "nightly"
	)
	var _ = Context("when creating a hosted cluster", func() {

		It("OK: Validates successfully a cluster for HyperShift with a supported version", func() {
			v, err := validateVersion("4.12.0", []string{"4.12.0"}, nightly, false, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("openshift-v4.12.0"))
		})
		It(`KO: Fails to validate a cluster for a hosted
		cluster when the user provides an unsupported version`,
			func() {
				v, err := validateVersion("4.11.5", []string{"4.11.5"}, nightly, false, true)
				Expect(err).To(BeEquivalentTo(fmt.Errorf("version '4.11.5' is not supported for hosted clusters")))
				Expect(v).To(BeEmpty())
			})
		It(`KO: Fails to validate a cluster for a hosted cluster
		when the user provides an invalid or malformed version`,
			func() {
				v, err := validateVersion("foo.bar", []string{"foo.bar"}, nightly, false, true)
				Expect(err).To(BeEquivalentTo(
					fmt.Errorf("error while parsing OCP version 'foo.bar': Malformed version: foo.bar")))
				Expect(v).To(BeEmpty())
			})

	})
})
