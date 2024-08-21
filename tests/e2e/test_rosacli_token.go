package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("rosacli token",
	labels.Feature.Token,
	func() {
		defer GinkgoRecover()

		var (
			rosaClient *rosacli.Client
		)

		BeforeEach(func() {
			By("Init the client")
			rosaClient = rosacli.NewClient()
		})

		It("Generate token via `rosa token` command and check all flags of it - [id:72162]",
			labels.High, labels.Runtime.OCMResources,
			func() {
				ocmService := rosaClient.OCMResource
				By("check `rosa token --header`")
				headerOutput, err := ocmService.Token("--header")
				Expect(err).To(BeNil())
				Expect(headerOutput.String()).ToNot(BeEmpty())

				By("Generate new token")
				originalToken, err := ocmService.Token()
				Expect(err).To(BeNil())
				newGeneratedToken, err := ocmService.Token("--generate")
				Expect(err).To(BeNil())
				Expect(originalToken.String()).ToNot(Equal(newGeneratedToken.String()))

				By("Check `rosa token --parload`")
				payloadOutput, err := ocmService.Token("--payload")
				Expect(err).To(BeNil())
				Expect(payloadOutput.String()).ToNot(BeEmpty())

				By("Checlk `rosa token --signature`")
				signatureOutput, err := ocmService.Token("--signature")
				Expect(err).To(BeNil())
				Expect(signatureOutput.String()).ToNot(BeEmpty())
			})
	})
