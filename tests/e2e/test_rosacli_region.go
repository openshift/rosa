package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Region",
	labels.Day2,
	labels.FeatureRegion,
	func() {
		defer GinkgoRecover()

		var (
			clusterID          string
			rosaClient         *rosacli.Client
			ocmResourceService rosacli.OCMResourceService
		)

		BeforeEach(func() {

			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			ocmResourceService = rosaClient.OCMResource
		})

		It("can list regions - [id:55729]",
			labels.High,
			func() {

				By("List region")
				usersTabNonH, _, err := ocmResourceService.ListRegion()
				Expect(err).To(BeNil())
				Expect(len(usersTabNonH)).NotTo(Equal(0))

				By("List region --hosted-cp")
				usersTabH, _, err := ocmResourceService.ListRegion("--hosted-cp")
				Expect(err).To(BeNil())
				Expect(len(usersTabH)).NotTo(Equal(0))

				By("Check out of 'rosa list region --hosted-cp' are supported for hosted-cp clusters")
				for _, r := range usersTabH {
					Expect(r.MultiAZSupported).To(Equal("true"))
				}
			})
	})
