package e2e

import (
	//nolint:staticcheck
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/helper"
)

var _ = Describe("DNS domain tests",
	labels.Feature.Ingress,
	func() {
		defer GinkgoRecover()

		var (
			rosaClient       *rosacli.Client
			dnsDomainService rosacli.OCMResourceService
			dnsDomainC       string
			dnsDomainH       string
		)

		BeforeEach(func() {

			By("Init the client")
			rosaClient = rosacli.NewClient()
			dnsDomainService = rosaClient.OCMResource

		})

		It("can create/list/delete dns-domain via rosacli - [id:65793]",
			labels.Critical, labels.Runtime.OCMResources, labels.Hyperfleet.Validated,
			func() {
				isV2 := isHyperfleetMode()
				defaultArchitecture := "classic"
				if isV2 {
					defaultArchitecture = "hcp"
				}
				By("Create dns-domain using the default architecture")
				outputC, err := dnsDomainService.CreateDNSDomain()
				Expect(err).ToNot(HaveOccurred())
				dnsDomainC, err = helper.ExtractDNSDomainID(outputC)
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() {
					By("Delete the created dns-domain using the default architecture")
					output, err := dnsDomainService.DeleteDNSDomain(dnsDomainC)
					Expect(err).ToNot(HaveOccurred())
					Expect(output.String()).To(ContainSubstring("Successfully deleted dns domain '%s'", dnsDomainC))

					By("Check the created dns-domain using the default architecture delete")
					out, err := dnsDomainService.ListDNSDomain()
					Expect(err).ToNot(HaveOccurred())
					Expect(out.String()).ToNot(ContainSubstring(dnsDomainC))
				})

				By("List the created dns-domain using the default architecture")
				out, err := dnsDomainService.ListDNSDomain()
				Expect(err).ToNot(HaveOccurred())
				dnsDomainList, err := dnsDomainService.ReflectDNSDomainList(out)
				Expect(err).ToNot(HaveOccurred())
				dnsDomain := dnsDomainList.GetDNSDomain(dnsDomainC)
				Expect(dnsDomain.ID).To(Equal(dnsDomainC))
				Expect(dnsDomain.UserDefined).To(Equal("Yes"))
				Expect(dnsDomain.Architecture).To(Equal(defaultArchitecture))

				By("Create dns-domain for hosted-cp cluster")
				outputH, err := dnsDomainService.CreateDNSDomain("--hosted-cp")
				Expect(err).ToNot(HaveOccurred())
				dnsDomainH, err = helper.ExtractDNSDomainID(outputH)
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() {
					By("Delete the created dns-domain for hosted-cp cluster")
					output, err := dnsDomainService.DeleteDNSDomain(dnsDomainH)
					Expect(err).ToNot(HaveOccurred())
					Expect(output.String()).To(ContainSubstring("Successfully deleted dns domain '%s'", dnsDomainH))

					By("Check the created dns-domain for hosted-cp cluster delete")
					out, err := dnsDomainService.ListDNSDomain()
					Expect(err).ToNot(HaveOccurred())
					Expect(out.String()).ToNot(ContainSubstring(dnsDomainH))
				})

				By("List the created dns-domain for hosted-cp cluster")
				out, err = dnsDomainService.ListDNSDomain()
				Expect(err).ToNot(HaveOccurred())
				dnsDomainList, err = dnsDomainService.ReflectDNSDomainList(out)
				Expect(err).ToNot(HaveOccurred())
				dnsDomain = dnsDomainList.GetDNSDomain(dnsDomainH)
				Expect(dnsDomain.ID).To(Equal(dnsDomainH))
				Expect(dnsDomain.UserDefined).To(Equal("Yes"))
				Expect(dnsDomain.Architecture).To(Equal("hcp"))

				By("List the created dns-domain with '--hosted-cp' flag")
				out, err = dnsDomainService.ListDNSDomain("--hosted-cp")
				Expect(err).ToNot(HaveOccurred())
				dnsDomainList, err = dnsDomainService.ReflectDNSDomainList(out)
				Expect(err).ToNot(HaveOccurred())
				dnsDomain = dnsDomainList.GetDNSDomain(dnsDomainH)
				Expect(dnsDomain.ID).To(Equal(dnsDomainH))
				Expect(dnsDomain.UserDefined).To(Equal("Yes"))
				Expect(dnsDomain.Architecture).To(Equal("hcp"))
				dnsDomain = dnsDomainList.GetDNSDomain(dnsDomainC)
				if isV2 {
					Expect(dnsDomain.ID).To(Equal(dnsDomainC))
					Expect(dnsDomain.UserDefined).To(Equal("Yes"))
					Expect(dnsDomain.Architecture).To(Equal("hcp"))
				} else {
					Expect(dnsDomain.ID).To(BeEmpty())
					Expect(dnsDomain.UserDefined).To(BeEmpty())
					Expect(dnsDomain.Architecture).To(BeEmpty())
				}

				By("List all created dns-domains for all supported architectures")
				out, err = dnsDomainService.ListDNSDomain("--all")
				Expect(err).ToNot(HaveOccurred())
				dnsDomainList, err = dnsDomainService.ReflectDNSDomainList(out)
				Expect(err).ToNot(HaveOccurred())
				dnsDomain = dnsDomainList.GetDNSDomain(dnsDomainH)
				Expect(dnsDomain.ID).To(Equal(dnsDomainH))
				Expect(dnsDomain.UserDefined).To(Equal("Yes"))
				Expect(dnsDomain.Architecture).To(Equal("hcp"))
				dnsDomain = dnsDomainList.GetDNSDomain(dnsDomainC)
				Expect(dnsDomain.ID).To(Equal(dnsDomainC))
				Expect(dnsDomain.UserDefined).To(Equal("Yes"))
				Expect(dnsDomain.Architecture).To(Equal(defaultArchitecture))

				By("List all created dns-domains with '--hosted-cp' flag")
				out, err = dnsDomainService.ListDNSDomain("--all", "--hosted-cp")
				Expect(err).ToNot(HaveOccurred())
				dnsDomainList, err = dnsDomainService.ReflectDNSDomainList(out)
				Expect(err).ToNot(HaveOccurred())
				dnsDomain = dnsDomainList.GetDNSDomain(dnsDomainH)
				Expect(dnsDomain.ID).To(Equal(dnsDomainH))
				Expect(dnsDomain.UserDefined).To(Equal("Yes"))
				Expect(dnsDomain.Architecture).To(Equal("hcp"))
				dnsDomain = dnsDomainList.GetDNSDomain(dnsDomainC)
				if isV2 {
					Expect(dnsDomain.ID).To(Equal(dnsDomainC))
					Expect(dnsDomain.UserDefined).To(Equal("Yes"))
					Expect(dnsDomain.Architecture).To(Equal("hcp"))
				} else {
					Expect(dnsDomain.ID).To(BeEmpty())
					Expect(dnsDomain.UserDefined).To(BeEmpty())
					Expect(dnsDomain.Architecture).To(BeEmpty())
				}

				By("Verify the scope of the --all flag")
				out, err = dnsDomainService.ListDNSDomain("--all")
				Expect(err).ToNot(HaveOccurred())
				dnsDomainList, err = dnsDomainService.ReflectDNSDomainList(out)
				Expect(err).ToNot(HaveOccurred())
				allDomains := dnsDomainList.DNSDomainList
				out, err = dnsDomainService.ListDNSDomain()
				Expect(err).ToNot(HaveOccurred())
				dnsDomainList, err = dnsDomainService.ReflectDNSDomainList(out)
				Expect(err).ToNot(HaveOccurred())
				if isV2 {
					Expect(allDomains).To(ConsistOf(dnsDomainList.DNSDomainList))
				} else {
					Expect(len(allDomains)).To(BeNumerically(">", len(dnsDomainList.DNSDomainList)))
				}
			})

	})
