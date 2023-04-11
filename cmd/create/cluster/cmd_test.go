package cluster

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
)

var _ = Describe("Validates OCP version", func() {

	const (
		nightly   = "nightly"
		stable    = "stable"
		candidate = "candidate"
		fast      = "fast"
	)
	var client *ocm.Client
	BeforeEach(func() {
		c, err := ocm.NewClient().Logger(logging.NewLogger()).Build()
		Expect(err).NotTo(HaveOccurred())
		client = c
	})

	var _ = Context("when creating a hosted cluster", func() {

		It("OK: Validates successfully a cluster for hosted clusters with a supported version", func() {
			v, err := client.ValidateVersion("4.12.5", []string{"4.12.5"}, stable, false, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("openshift-v4.12.5"))
		})

		It("OK: Validates successfully a nightly version of OCP for hosted clusters "+
			"with a supported version", func() {
			v, err := client.ValidateVersion("4.12.0-0.nightly-2023-04-10-222146",
				[]string{"4.12.0-0.nightly-2023-04-10-222146"}, nightly, false, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("openshift-v4.12.0-0.nightly-2023-04-10-222146-nightly"))
		})

		It("KO: Fails with a nightly version of OCP for hosted clusters "+
			"in a not supported version", func() {
			v, err := client.ValidateVersion("4.11.0-0.nightly-2022-10-17-040259",
				[]string{"4.11.0-0.nightly-2022-10-17-040259"}, nightly, false, true)
			Expect(err).To(BeEquivalentTo(
				fmt.Errorf("version '4.11.0-0.nightly-2022-10-17-040259' " +
					"is not supported for hosted clusters")))
			Expect(v).To(Equal(""))
		})

		It("OK: Validates successfully the next major release of OCP for hosted clusters "+
			"with a supported version", func() {
			v, err := client.ValidateVersion("4.13.0-rc.2", []string{"4.13.0-rc.2"}, candidate, false, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("openshift-v4.13.0-rc.2-candidate"))
		})

		It(`KO: Fails to validate a cluster for a hosted
		cluster when the user provides an unsupported version`, func() {
			v, err := client.ValidateVersion("4.11.5", []string{"4.11.5"}, stable, false, true)
			Expect(err).To(BeEquivalentTo(fmt.Errorf("version '4.11.5' is not supported for hosted clusters")))
			Expect(v).To(BeEmpty())
		})

		It(`KO: Fails to validate a cluster for a hosted cluster
		when the user provides an invalid or malformed version`, func() {
			v, err := client.ValidateVersion("foo.bar", []string{"foo.bar"}, stable, false, true)
			Expect(err).To(BeEquivalentTo(
				fmt.Errorf("version 'foo.bar' was not found")))
			Expect(v).To(BeEmpty())
		})
	})
	var _ = Context("when creating a classic cluster", func() {
		It("OK: Validates successfully a cluster with a supported version", func() {
			v, err := client.ValidateVersion("4.11.0", []string{"4.11.0"}, stable, true, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("openshift-v4.11.0"))
		})
	})
})
