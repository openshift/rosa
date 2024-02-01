package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/aws/mocks"
)

var _ = Describe("Is Account Role Version Compatible", func() {
	When("Role isn't an account role", func() {
		It("Should return not compatible", func() {
			isCompatible, err := isAccountRoleVersionCompatible([]iamtypes.Tag{}, InstallerAccountRole, "4.14")
			Expect(err).To(BeNil())
			Expect(isCompatible).To(Equal(false))
		})
	})
	When("Role OCP version isn't compatible", func() {
		It("Should return not compatible", func() {
			tagsList := []iamtypes.Tag{
				{
					Key:   aws.String("rosa_openshift_version"),
					Value: aws.String("4.13"),
				},
			}
			isCompatible, err := isAccountRoleVersionCompatible(tagsList, InstallerAccountRole, "4.14")
			Expect(err).To(BeNil())
			Expect(isCompatible).To(Equal(false))
		})
	})
	When("Role version is compatible", func() {
		It("Should return compatible", func() {
			tagsList := []iamtypes.Tag{
				{
					Key:   aws.String("rosa_openshift_version"),
					Value: aws.String("4.14"),
				},
			}
			isCompatible, err := isAccountRoleVersionCompatible(tagsList, InstallerAccountRole, "4.14")
			Expect(err).To(BeNil())
			Expect(isCompatible).To(Equal(true))
		})
	})
	When("Role has managed policies, ignores openshift version", func() {
		It("Should return compatible", func() {
			tagsList := []iamtypes.Tag{
				{
					Key:   aws.String("rosa_openshift_version"),
					Value: aws.String("4.12"),
				},
				{
					Key:   aws.String("rosa_managed_policies"),
					Value: aws.String("true"),
				},
			}
			isCompatible, err := isAccountRoleVersionCompatible(tagsList, InstallerAccountRole, "4.14")
			Expect(err).To(BeNil())
			Expect(isCompatible).To(Equal(true))
		})
	})
	When("Role has HCP managed policies when trying to create classic cluster", func() {
		It("Should return incompatible", func() {
			tagsList := []iamtypes.Tag{
				{
					Key:   aws.String("rosa_openshift_version"),
					Value: aws.String("4.12"),
				},
				{
					Key:   aws.String("rosa_managed_policies"),
					Value: aws.String("true"),
				},
				{
					Key:   aws.String("rosa_hcp_policies"),
					Value: aws.String("true"),
				},
			}
			isCompatible, err := validateAccountRoleVersionCompatibilityClassic(InstallerAccountRole, "4.12",
				tagsList)
			Expect(err).To(BeNil())
			Expect(isCompatible).To(Equal(false))
		})
	})
	When("Role has classic policies when trying to create an HCP cluster", func() {
		It("Should return incompatible", func() {
			tagsList := []iamtypes.Tag{
				{
					Key:   aws.String("rosa_openshift_version"),
					Value: aws.String("4.12"),
				},
				{
					Key:   aws.String("rosa_managed_policies"),
					Value: aws.String("true"),
				},
			}
			isCompatible, err := validateAccountRoleVersionCompatibilityHostedCp(InstallerAccountRole, "4.12",
				tagsList)
			Expect(err).To(BeNil())
			Expect(isCompatible).To(Equal(false))
		})
	})
})

var _ = Describe("DeleteRole Validation", func() {

	var (
		client     awsClient
		mockIamAPI *mocks.MockIAMAPI
		mockCtrl   *gomock.Controller
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockIamAPI = mocks.NewMockIAMAPI(mockCtrl)
		client = awsClient{
			iamClient: mockIamAPI,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})
	When("Role doesn't exist", func() {
		It("Should return NoSuchEntityException", func() {
			role := "test"
			expectedErrorMessage := fmt.Sprintf("operator role '%s' does not exist. Skipping", role)
			mockIamAPI.EXPECT().DeleteRole(&iam.DeleteRoleInput{
				RoleName: aws.String(role),
			}).Return(nil, awserr.New(iam.ErrCodeNoSuchEntityException, "Role does not exist", nil))

			err := client.DeleteRole(role)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expectedErrorMessage))
		})
	})
	When("Role exists", func() {
		It("Should delete the role successfully", func() {
			role := "test"
			mockIamAPI.EXPECT().DeleteRole(&iam.DeleteRoleInput{
				RoleName: aws.String(role),
			}).Return(&iam.DeleteRoleOutput{}, nil)
			err := client.DeleteRole(role)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
