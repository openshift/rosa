package e2e

import (
	"strings"

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
				parsedHeaderOutput, err := rosaClient.Runner.RunPipeline(
					[]string{"rosa", "token"},
					[]string{"cut", "-d", ".", "-f", "1"},
					[]string{"tr", "_-", "/+"},
					[]string{"base64", "-d"},
				)
				Expect(err).To(BeNil())
				headerOutput, err := ocmService.Token("--header")
				Expect(err).To(BeNil())

				Expect(headerOutput.String()).To(ContainSubstring(parsedHeaderOutput.String()))

				By("Generate new token")
				originalToken, err := ocmService.Token()
				Expect(err).To(BeNil())
				newGeneratedToken, err := ocmService.Token("--generate")
				Expect(err).To(BeNil())
				Expect(originalToken.String()).ToNot(Equal(newGeneratedToken.String()))

				By("Check `rosa token --parload`")
				parsedPayloadOutput, err := rosaClient.Runner.RunPipeline(
					[]string{"rosa", "token"},
					[]string{"cut", "-d", ".", "-f", "2"},
					[]string{"tr", "_-", "/+"},
					[]string{"base64", "-d"},
				)
				Expect(err).To(BeNil())
				payloadOutput, err := ocmService.Token("--payload")
				Expect(err).To(BeNil())
				Expect(strings.TrimSpace(payloadOutput.String())).
					To(Equal(
						strings.TrimSpace(parsedPayloadOutput.String())))

				By("Checlk `rosa token --signature`")
				parsedSignatureOutput, err := rosaClient.Runner.RunPipeline(
					[]string{"rosa", "token"},
					[]string{"cut", "-d", ".", "-f", "3"},
					[]string{"tr", "_-", "/+"},
					[]string{"base64", "-d"},
				)
				Expect(err).To(BeNil())
				signatureOutput, err := ocmService.Token("--signature")
				Expect(err).To(BeNil())
				Expect(strings.TrimSpace(signatureOutput.String())).
					To(ContainSubstring(
						strings.TrimSpace(parsedSignatureOutput.String())))
			})
	})
