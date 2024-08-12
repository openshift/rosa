package e2e

import (
	"fmt"
	"time"

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
			rosaClient             *rosacli.Client
			ocmResourceService     rosacli.OCMResourceService
			permissionsBoundaryArn string = "arn:aws:iam::aws:policy/AdministratorAccess"
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

		It("List instance-types with region flag - [id:72174]",
			labels.Low, labels.Runtime.OCMResources,
			func() {
				By("List the available instance-types with the region flag")
				typesList := []string{"dl1.24xlarge", "g4ad.16xlarge", "c5.xlarge"}
				region := "us-west-2"
				accountRolePrefix := fmt.Sprintf("QEAuto-accr72174-%s", time.Now().UTC().Format("20060102"))
				_, err := ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", accountRolePrefix,
					"--permissions-boundary", permissionsBoundaryArn,
					"-y")
				Expect(err).To(BeNil())
				defer ocmResourceService.DeleteAccountRole("--mode", "auto", "--prefix", accountRolePrefix, "-y")

				accountRoleList, _, err := ocmResourceService.ListAccountRole()
				Expect(err).To(BeNil())
				classicInstallerRoleArn := accountRoleList.InstallerRole(accountRolePrefix, false).RoleArn
				availableMachineTypes, _, err := ocmResourceService.ListInstanceTypes(
					"--region", region, "--role-arn", classicInstallerRoleArn)
				Expect(err).To(BeNil())
				var availableMachineTypesIDs []string
				for _, it := range availableMachineTypes.InstanceTypesList {
					availableMachineTypesIDs = append(availableMachineTypesIDs, it.ID)
				}
				Expect(availableMachineTypesIDs).To(ContainElements(typesList))

				By("List the available instance-types with the region flag and hosted-cp flag")
				availableMachineTypes, _, err = ocmResourceService.ListInstanceTypes(
					"--region", region, "--role-arn", classicInstallerRoleArn, "--hosted-cp")
				Expect(err).To(BeNil())
				for _, it := range availableMachineTypes.InstanceTypesList {
					availableMachineTypesIDs = append(availableMachineTypesIDs, it.ID)
				}
				Expect(availableMachineTypesIDs).To(ContainElements(typesList))

				By("Try to list instance-types with invalid region")
				availableMachineTypes, output, err := ocmResourceService.ListInstanceTypes(
					"--region", "xxx", "--role-arn", classicInstallerRoleArn)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("ERR: Unsupported region 'xxx', available regions"))
			})
	})
