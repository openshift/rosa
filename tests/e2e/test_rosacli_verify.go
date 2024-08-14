package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("rosacli verify subcommand",
	labels.Feature.VerifyResources,
	func() {
		defer GinkgoRecover()

		var (
			rosaClient    *rosacli.Client
			verifyService rosacli.VerifyService
		)

		BeforeEach(func() {
			By("Init the client")
			rosaClient = rosacli.NewClient()
			verifyService = rosaClient.Verify

		})

		It("to verify oc/permission/quota/rosa client via rosacli - [id:38851]",
			labels.Medium, labels.Runtime.OCMResources,
			func() {

				By("Verify openshift client")
				_, err := verifyService.VerifyOC()
				Expect(err).ToNot(HaveOccurred())

				By("Verify permissions")
				out, err := verifyService.VerifyPermissions()
				Expect(err).ToNot(HaveOccurred())
				Expect(out.String()).To(ContainSubstring("AWS SCP policies ok"))

				By("Verify quota")
				_, err = verifyService.VerifyQuota()
				Expect(err).ToNot(HaveOccurred())

				By("Verify rosacli client")
				_, err = verifyService.VerifyRosaClient()
				Expect(err).ToNot(HaveOccurred())
			})
	})
