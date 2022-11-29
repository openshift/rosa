package cluster

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validates OCP version", func() {

	const (
		nightly = "nightly"
		stable  = "stable"
		fast    = "fast"
	)
	var _ = Context("when creating a hosted cluster", func() {

		It("OK: Validates successfully a cluster for hosted clusters with a supported version", func() {
			v, err := validateVersion("4.12.0", []string{"4.12.0"}, stable, false, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("openshift-v4.12.0"))
		})

		It("OK: Validates successfully a nightly version of OCP for hosted clusters "+
			"with a supported version", func() {
			v, err := validateVersion("4.12.0-0.nightly-2022-11-25-185455-nightly",
				[]string{"4.12.0-0.nightly-2022-11-25-185455-nightly"}, nightly, false, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("openshift-v4.12.0-0.nightly-2022-11-25-185455-nightly"))
		})

		It("KO: Fails with a nightly version of OCP for hosted clusters "+
			"in a not supported version", func() {
			v, err := validateVersion("4.11.0-0.nightly-2022-10-17-040259-nightly",
				[]string{"4.11.0-0.nightly-2022-10-17-040259-nightly"}, nightly, false, true)
			Expect(err).To(BeEquivalentTo(
				fmt.Errorf("version '4.11.0-0.nightly-2022-10-17-040259-nightly' " +
					"is not supported for hosted clusters")))
			Expect(v).To(Equal(""))
		})

		It("OK: Validates successfully the next major release of OCP for hosted clusters "+
			"with a supported version", func() {
			v, err := validateVersion("4.13.0", []string{"4.13.0"}, fast, false, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("openshift-v4.13.0"))
		})

		It(`KO: Fails to validate a cluster for a hosted
		cluster when the user provides an unsupported version`, func() {
			v, err := validateVersion("4.11.5", []string{"4.11.5"}, stable, false, true)
			Expect(err).To(BeEquivalentTo(fmt.Errorf("version '4.11.5' is not supported for hosted clusters")))
			Expect(v).To(BeEmpty())
		})

		It(`KO: Fails to validate a cluster for a hosted cluster
		when the user provides an invalid or malformed version`, func() {
			v, err := validateVersion("foo.bar", []string{"foo.bar"}, stable, false, true)
			Expect(err).To(BeEquivalentTo(
				fmt.Errorf("error while parsing OCP version 'foo.bar': Malformed version: foo.bar")))
			Expect(v).To(BeEmpty())
		})
	})
	var _ = Context("when creating a classic cluster", func() {

		It("OK: Validates successfully a cluster with a supported version", func() {
			v, err := validateVersion("4.11.0", []string{"4.11.0"}, stable, true, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("openshift-v4.11.0"))
		})
	})
})
