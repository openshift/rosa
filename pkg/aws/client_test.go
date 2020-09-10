package aws_test

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/openshift/moactl/pkg/aws"
	"github.com/openshift/moactl/pkg/aws/mocks"
)

var _ = Describe("Client", func() {
	var (
		client   aws.Client
		mockCtrl *gomock.Controller

		mockCfAPI  *mocks.MockCloudFormationAPI
		mockIamAPI *mocks.MockIAMAPI
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCfAPI = mocks.NewMockCloudFormationAPI(mockCtrl)
		mockIamAPI = mocks.NewMockIAMAPI(mockCtrl)
		client = aws.New(
			logrus.New(),
			mockIamAPI,
			mocks.NewMockOrganizationsAPI(mockCtrl),
			mocks.NewMockSTSAPI(mockCtrl),
			mockCfAPI,
			mocks.NewMockServiceQuotasAPI(mockCtrl),
			&session.Session{},
		)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("ValidateCFUserCredentials", func() {
		var (
			CFsecretID      = "sut"
			IAMAccessKey    = "sut"
			oddIAMAccessKey = "longtestkey"
			status          = "Active"
		)
		Context("when creds are OK and matches", func() {
			BeforeEach(func() {
				mockCfAPI.EXPECT().DescribeStackResource(gomock.Any()).Return(&cloudformation.DescribeStackResourceOutput{
					StackResourceDetail: &cloudformation.StackResourceDetail{
						PhysicalResourceId: &CFsecretID,
					},
				}, nil)

				mockIamAPI.EXPECT().ListAccessKeys(gomock.Any()).Return(&iam.ListAccessKeysOutput{
					AccessKeyMetadata: []*iam.AccessKeyMetadata{
						{
							AccessKeyId: &IAMAccessKey,
							Status:      &status,
						},
					},
				}, nil)
			})
			It("should finish successfully and return nil", func() {
				err := client.ValidateCFUserCredentials()
				Expect(err).To(BeNil())
			})
		})

		Context("when the credentials don't match", func() {
			BeforeEach(func() {
				mockCfAPI.EXPECT().DescribeStackResource(gomock.Any()).Return(&cloudformation.DescribeStackResourceOutput{
					StackResourceDetail: &cloudformation.StackResourceDetail{
						PhysicalResourceId: &CFsecretID,
					},
				}, nil)

				mockIamAPI.EXPECT().ListAccessKeys(gomock.Any()).Return(&iam.ListAccessKeysOutput{
					AccessKeyMetadata: []*iam.AccessKeyMetadata{
						{
							AccessKeyId: &oddIAMAccessKey,
						},
					},
				}, nil)
			})
			It("should return err", func() {
				err := client.ValidateCFUserCredentials()
				Expect(err.Error()).Should(ContainSubstring("Invalid CloudFormation stack credentials"))
			})
		})

	})
	Context("EnsureOsdCcsAdminUser", func() {
		var (
			stackName   string
			stackStatus string
		)
		BeforeEach(func() {
			stackName = "fake-stack"
		})
		Context("When the cloudformation stack already exists", func() {
			JustBeforeEach(func() {
				mockCfAPI.EXPECT().ListStacks(gomock.Any()).Return(&cloudformation.ListStacksOutput{
					StackSummaries: []*cloudformation.StackSummary{
						{
							StackName:   &stackName,
							StackStatus: &stackStatus,
						},
					},
				}, nil)
			})

			Context("When stack is in CREATE_COMPLETE state", func() {
				BeforeEach(func() {
					stackStatus = cloudformation.StackStatusCreateComplete
				})
				It("Returns without error", func() {
					stackCreated, err := client.EnsureOsdCcsAdminUser(stackName)

					Expect(stackCreated).To(BeFalse())
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("When stack is in DELETE_COMPLETE state", func() {
				BeforeEach(func() {
					stackStatus = cloudformation.StackStatusDeleteComplete
					mockCfAPI.EXPECT().CreateStack(gomock.Any()).Return(nil, nil)
					mockCfAPI.EXPECT().WaitUntilStackCreateComplete(gomock.Any()).Return(nil)
				})
				It("Creates a cloudformation stack", func() {
					stackCreated, err := client.EnsureOsdCcsAdminUser(stackName)

					Expect(stackCreated).To(BeTrue())
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("When stack is in ROLLBACK_COMPLETE state", func() {
				BeforeEach(func() {
					stackStatus = cloudformation.StackStatusRollbackComplete
				})

				It("Returns error telling the stack is in an invalid state", func() {
					stackCreated, err := client.EnsureOsdCcsAdminUser(stackName)

					Expect(stackCreated).To(BeFalse())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(
						"exists with status ROLLBACK_COMPLETE. Expected status is CREATE_COMPLETE"))
				})
			})
		})

		Context("When the cloudformation stack does not exists", func() {
			BeforeEach(func() {
				mockCfAPI.EXPECT().ListStacks(gomock.Any()).Return(&cloudformation.ListStacksOutput{
					StackSummaries: []*cloudformation.StackSummary{},
				}, nil)
				mockCfAPI.EXPECT().CreateStack(gomock.Any()).Return(nil, nil)
				mockCfAPI.EXPECT().WaitUntilStackCreateComplete(gomock.Any()).Return(nil)
			})

			It("Creates a cloudformation stack", func() {
				stackCreated, err := client.EnsureOsdCcsAdminUser(stackName)

				Expect(err).NotTo(HaveOccurred())
				Expect(stackCreated).To(BeTrue())
			})
		})
	})
})
