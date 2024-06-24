package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Region",
	labels.Feature.Regions,
	func() {
		defer GinkgoRecover()

		var (
			rosaClient         *rosacli.Client
			ocmResourceService rosacli.OCMResourceService
		)

		BeforeEach(func() {
			By("Init the client")
			rosaClient = rosacli.NewClient()
			ocmResourceService = rosaClient.OCMResource
		})

		It("can list regions - [id:55729]",
			labels.High, labels.Runtime.OCMResources,
			func() {

				By("List region")
				regionList, err := ocmResourceService.List().Regions().ToStruct()
				Expect(err).To(BeNil())
				Expect(len(regionList.([]*rosacli.CloudRegion))).NotTo(Equal(0))

				By("List region --hosted-cp")
				hcpRegionList, err := ocmResourceService.List().Regions().Parameters("--hosted-cp").ToStruct()
				Expect(err).To(BeNil())
				Expect(len(hcpRegionList.([]*rosacli.CloudRegion))).NotTo(Equal(0))

				By("Check out of 'rosa list region --hosted-cp' are supported for hosted-cp clusters")
				for _, r := range hcpRegionList.([]*rosacli.CloudRegion) {
					Expect(r.MultiAZSupported).To(Equal("true"))
				}
			})
	})
