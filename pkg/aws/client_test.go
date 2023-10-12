package aws_test

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/openshift-online/ocm-sdk-go/helpers"

	awsSdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/mocks"
	rosaTags "github.com/openshift/rosa/pkg/aws/tags"
)

var _ = Describe("Client", func() {
	var (
		client   aws.Client
		mockCtrl *gomock.Controller

		mockEC2API            *mocks.MockEC2API
		mockCfAPI             *mocks.MockCloudFormationAPI
		mockIamAPI            *mocks.MockIAMAPI
		mockS3API             *mocks.MockS3API
		mockSecretsManagerAPI *mocks.MockSecretsManagerAPI
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCfAPI = mocks.NewMockCloudFormationAPI(mockCtrl)
		mockIamAPI = mocks.NewMockIAMAPI(mockCtrl)
		mockEC2API = mocks.NewMockEC2API(mockCtrl)
		mockS3API = mocks.NewMockS3API(mockCtrl)
		mockSecretsManagerAPI = mocks.NewMockSecretsManagerAPI(mockCtrl)
		client = aws.New(
			logrus.New(),
			mockIamAPI,
			mockEC2API,
			mocks.NewMockOrganizationsAPI(mockCtrl),
			mockS3API,
			mockSecretsManagerAPI,
			mocks.NewMockSTSAPI(mockCtrl),
			mockCfAPI,
			mocks.NewMockServiceQuotasAPI(mockCtrl),
			&session.Session{},
			&aws.AccessKey{},
			false,
		)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("EnsureOsdCcsAdminUser", func() {
		var (
			stackName     string
			stackStatus   string
			adminUserName string
		)
		BeforeEach(func() {
			stackName = "fake-stack"
			adminUserName = "fake-admin-username"
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
					mockIamAPI.EXPECT().GetUser(gomock.Any()).Return(
						&iam.GetUserOutput{User: &iam.User{UserName: &adminUserName}},
						awserr.New(iam.ErrCodeNoSuchEntityException, "", nil),
					)
					mockCfAPI.EXPECT().UpdateStack(gomock.Any()).Return(nil, nil)
					mockCfAPI.EXPECT().WaitUntilStackUpdateComplete(gomock.Any()).Return(nil)
				})
				It("Returns without error", func() {
					stackCreated, err := client.EnsureOsdCcsAdminUser(stackName, adminUserName, aws.DefaultRegion)

					Expect(stackCreated).To(BeFalse())
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("When stack is in DELETE_COMPLETE state", func() {
				BeforeEach(func() {
					stackStatus = cloudformation.StackStatusDeleteComplete
					mockIamAPI.EXPECT().ListUsers(gomock.Any()).Return(&iam.ListUsersOutput{Users: []*iam.User{}}, nil)
					mockIamAPI.EXPECT().TagUser(gomock.Any()).Return(&iam.TagUserOutput{}, nil)
					mockIamAPI.EXPECT().GetUser(gomock.Any()).Return(
						&iam.GetUserOutput{User: &iam.User{UserName: &adminUserName}},
						awserr.New(iam.ErrCodeNoSuchEntityException, "", nil),
					)
					mockCfAPI.EXPECT().CreateStack(gomock.Any()).Return(nil, nil)
					mockCfAPI.EXPECT().WaitUntilStackCreateComplete(gomock.Any()).Return(nil)
				})
				It("Creates a cloudformation stack", func() {
					stackCreated, err := client.EnsureOsdCcsAdminUser(stackName, adminUserName, aws.DefaultRegion)

					Expect(stackCreated).To(BeTrue())
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("When stack is in ROLLBACK_COMPLETE state", func() {
				BeforeEach(func() {
					stackStatus = cloudformation.StackStatusRollbackComplete
					mockIamAPI.EXPECT().GetUser(gomock.Any()).Return(
						&iam.GetUserOutput{User: &iam.User{UserName: &adminUserName}},
						awserr.New(iam.ErrCodeNoSuchEntityException, "", nil),
					)
				})

				It("Returns error telling the stack is in an invalid state", func() {
					stackCreated, err := client.EnsureOsdCcsAdminUser(stackName, adminUserName, aws.DefaultRegion)

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
				mockIamAPI.EXPECT().ListUsers(gomock.Any()).Return(&iam.ListUsersOutput{Users: []*iam.User{}}, nil)
				mockIamAPI.EXPECT().TagUser(gomock.Any()).Return(&iam.TagUserOutput{}, nil)
				mockIamAPI.EXPECT().GetUser(gomock.Any()).Return(
					&iam.GetUserOutput{User: &iam.User{UserName: &adminUserName}},
					awserr.New(iam.ErrCodeNoSuchEntityException, "", nil),
				)
				mockCfAPI.EXPECT().CreateStack(gomock.Any()).Return(nil, nil)
				mockCfAPI.EXPECT().WaitUntilStackCreateComplete(gomock.Any()).Return(nil)
			})

			It("Creates a cloudformation stack", func() {
				stackCreated, err := client.EnsureOsdCcsAdminUser(stackName, adminUserName, aws.DefaultRegion)

				Expect(err).NotTo(HaveOccurred())
				Expect(stackCreated).To(BeTrue())
			})
		})
		//		Context("When the IAM user already exists"), func() {
		//			BeforeEach(func() {

		//			}
	})
	Context("CheckAdminUserNotExisting", func() {
		var (
			adminUserName string
		)
		BeforeEach(func() {
			adminUserName = "fake-admin-username"
			mockIamAPI.EXPECT().ListUsers(gomock.Any()).Return(&iam.ListUsersOutput{
				Users: []*iam.User{
					{
						UserName: &adminUserName,
					},
				},
			}, nil)
		})
		Context("When admin user already exists", func() {
			It("returns an error", func() {
				err := client.CheckAdminUserNotExisting(adminUserName)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Error creating user: IAM user"))
			})
		})
		Context("When admin user does not exist", func() {
			var (
				secondFakeAdminUserName string
			)
			BeforeEach(func() {
				secondFakeAdminUserName = "second-fake-admin-username"
			})
			It("returns true", func() {
				err := client.CheckAdminUserNotExisting(secondFakeAdminUserName)

				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
	Context("Get Account Role By ARN", func() {

		var testArn = "arn:aws:iam::765374464689:role/test-Installer-Role"
		var testName = "test-Installer-Role"
		var tags = []*iam.Tag{
			{Key: awsSdk.String(rosaTags.ManagedPolicies), Value: awsSdk.String(rosaTags.True)},
			{Key: awsSdk.String(rosaTags.RoleType), Value: awsSdk.String(aws.InstallerAccountRole)},
		}

		It("Finds and Returns Account Role", func() {

			mockIamAPI.EXPECT().GetRole(gomock.Any()).Return(&iam.GetRoleOutput{
				Role: &iam.Role{
					Arn:      &testArn,
					RoleName: &testName,
				},
			}, nil)

			mockIamAPI.EXPECT().ListRoleTags(gomock.Any()).Return(&iam.ListRoleTagsOutput{
				Tags: tags,
			}, nil)

			mockIamAPI.EXPECT().ListRolePolicies(gomock.Any()).Return(&iam.ListRolePoliciesOutput{
				PolicyNames: make([]*string, 0),
			}, nil)

			role, err := client.GetAccountRoleByArn(testArn)

			Expect(err).NotTo(HaveOccurred())
			Expect(role).NotTo(BeNil())

			Expect(role.RoleName).To(Equal(testName))
			Expect(role.RoleARN).To(Equal(testArn))
			Expect(role.RoleType).To(Equal(aws.InstallerAccountRoleType))
		})

		It("Returns nil when No Role with ARN exists", func() {
			mockIamAPI.EXPECT().GetRole(gomock.Any()).Return(nil, fmt.Errorf("role Doesn't Exist"))

			role, err := client.GetAccountRoleByArn(testArn)

			Expect(role).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("Returns nil when the Role exists, but it is not an Account Role", func() {

			var roleName = "not-an-account-role"

			mockIamAPI.EXPECT().GetRole(gomock.Any()).Return(&iam.GetRoleOutput{
				Role: &iam.Role{
					Arn:      &testArn,
					RoleName: &roleName,
				},
			}, nil)

			role, err := client.GetAccountRoleByArn(testArn)
			Expect(role).To(BeNil())
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("List Subnets", func() {

		subnetOneId := "test-subnet-1"
		subnetTwoId := "test-subnet-2"
		subnet := ec2.Subnet{
			SubnetId: helpers.NewString(subnetOneId),
		}

		subnet2 := ec2.Subnet{
			SubnetId: helpers.NewString(subnetTwoId),
		}

		var subnets []*ec2.Subnet
		subnets = append(subnets, &subnet, &subnet2)

		It("Lists all", func() {

			var request *ec2.DescribeSubnetsInput

			mockEC2API.EXPECT().DescribeSubnets(gomock.Any()).DoAndReturn(
				func(arg *ec2.DescribeSubnetsInput) (*ec2.DescribeSubnetsOutput, error) {
					request = arg
					return &ec2.DescribeSubnetsOutput{
						Subnets: subnets,
					}, nil
				})

			subs, err := client.ListSubnets()
			Expect(err).NotTo(HaveOccurred())

			Expect(subs).To(HaveLen(2))
			Expect(request.SubnetIds).To(BeEmpty())
		})

		It("Lists by subnet ids", func() {

			var request *ec2.DescribeSubnetsInput

			mockEC2API.EXPECT().DescribeSubnets(gomock.Any()).DoAndReturn(
				func(arg *ec2.DescribeSubnetsInput) (*ec2.DescribeSubnetsOutput, error) {
					request = arg
					return &ec2.DescribeSubnetsOutput{
						Subnets: subnets,
					}, nil
				})

			subs, err := client.ListSubnets(subnetOneId, subnetTwoId)
			Expect(err).NotTo(HaveOccurred())

			Expect(subs).To(HaveLen(2))
			Expect(request.SubnetIds).To(ContainElements(&subnetOneId, &subnetTwoId))

		})
	})
})
