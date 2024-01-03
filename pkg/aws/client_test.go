package aws

import (
	"errors"
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/openshift-online/ocm-sdk-go/helpers"

	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cloudformationtypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	common "github.com/openshift-online/ocm-common/pkg/aws/validations"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/assets"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/mocks"
	rosaTags "github.com/openshift/rosa/pkg/aws/tags"
)

var _ = Describe("Client", func() {
	var (
		client   Client
		mockCtrl *gomock.Controller

		mockEC2API            *mocks.MockEc2ApiClient
		mockCfAPI             *mocks.MockCloudFormationApiClient
		mockIamAPI            *mocks.MockIamApiClient
		mockS3API             *mocks.MockS3ApiClient
		mockSecretsManagerAPI *mocks.MockSecretsManagerApiClient
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCfAPI = mocks.NewMockCloudFormationApiClient(mockCtrl)
		mockIamAPI = mocks.NewMockIamApiClient(mockCtrl)
		mockEC2API = mocks.NewMockEc2ApiClient(mockCtrl)
		mockS3API = mocks.NewMockS3ApiClient(mockCtrl)
		mockSecretsManagerAPI = mocks.NewMockSecretsManagerApiClient(mockCtrl)
		client = aws.New(
			awsSdk.Config{},
			logrus.New(),
			mockIamAPI,
			mockEC2API,
			mocks.NewMockOrganizationsApiClient(mockCtrl),
			mockS3API,
			mockSecretsManagerAPI,
			mocks.NewMockStsApiClient(mockCtrl),
			mockCfAPI,
			mocks.NewMockServiceQuotasApiClient(mockCtrl),
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
				mockCfAPI.EXPECT().ListStacks(context.Background(),
					&cloudformation.ListStacksInput{}).Return(
					&cloudformation.ListStacksOutput{
						StackSummaries: []cloudformationtypes.StackSummary{
							{
								StackName:   &stackName,
								StackStatus: cloudformationtypes.StackStatus(stackStatus),
							},
						},
					}, nil)
			})

			Context("When stack is in CREATE_COMPLETE state", func() {
				BeforeEach(func() {
					stackStatus = string(cloudformationtypes.StackStatusCreateComplete)
					cfTemplatePath := "templates/cloudformation/iam_user_osdCcsAdmin.json"
					cfTemplate, err := assets.Asset(cfTemplatePath)
					Expect(err).ToNot(HaveOccurred())
					cfTemplateBody := string(cfTemplate)
					mockIamAPI.EXPECT().GetUser(context.Background(),
						&iam.GetUserInput{UserName: &adminUserName}).Return(
						&iam.GetUserOutput{User: &iamtypes.User{UserName: &adminUserName}},
						&iamtypes.NoSuchEntityException{},
					)
					describeStacksOutput := &cloudformation.DescribeStacksOutput{
						Stacks: []cloudformationtypes.Stack{
							{
								StackName:   &stackName,
								StackStatus: cloudformationtypes.StackStatusCreateComplete,
							},
						},
					}

					mockCfAPI.EXPECT().
						DescribeStacks(gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(_ context.Context, _ *cloudformation.DescribeStacksInput,
							_ ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
							return describeStacksOutput, nil
						}).AnyTimes()
					mockCfAPI.EXPECT().
						UpdateStack(gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(_ context.Context, input *cloudformation.UpdateStackInput,
							_ ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error) {
							// Verify that the input parameters are as expected
							if *input.StackName != stackName {
								return nil, fmt.Errorf("unexpected stack name: got %s, want %s", *input.StackName, stackName)
							}
							if *input.TemplateBody == cfTemplateBody {
								// Simulate the error returned by AWS when no updates are to be performed
								return nil, &smithy.GenericAPIError{
									Code:    "ValidationError",
									Message: "No updates are to be performed.",
								}
							}
							return &cloudformation.UpdateStackOutput{
								StackId: &stackName,
							}, nil
						})
				})
				It("Returns without error", func() {
					stackCreated, err := client.EnsureOsdCcsAdminUser(stackName, adminUserName, DefaultRegion)

					Expect(stackCreated).To(BeFalse())
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("When stack is in DELETE_COMPLETE state", func() {
				BeforeEach(func() {
					stackStatus = string(cloudformationtypes.StackStatusDeleteComplete)
					mockIamAPI.EXPECT().ListUsers(context.Background(), gomock.Any()).Return(
						&iam.ListUsersOutput{Users: []iamtypes.User{}}, nil)
					mockIamAPI.EXPECT().TagUser(context.Background(), gomock.Any()).Return(&iam.TagUserOutput{}, nil)
					mockIamAPI.EXPECT().GetUser(context.Background(), &iam.GetUserInput{UserName: &adminUserName}).Return(
						&iam.GetUserOutput{User: &iamtypes.User{UserName: &adminUserName}},
						&iamtypes.NoSuchEntityException{},
					)
					describeStacksOutput := &cloudformation.DescribeStacksOutput{
						Stacks: []cloudformationtypes.Stack{
							{
								StackName:   &stackName,
								StackStatus: cloudformationtypes.StackStatusCreateComplete,
							},
						},
					}
					mockCfAPI.EXPECT().
						DescribeStacks(gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(_ context.Context, _ *cloudformation.DescribeStacksInput,
							_ ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
							return describeStacksOutput, nil
						}).AnyTimes()
					mockCfAPI.EXPECT().CreateStack(context.Background(), gomock.Any()).Return(nil, nil)
				})
				It("Creates a cloudformation stack", func() {
					stackCreated, err := client.EnsureOsdCcsAdminUser(stackName, adminUserName, DefaultRegion)

					Expect(stackCreated).To(BeTrue())
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("When stack is in ROLLBACK_COMPLETE state", func() {
				BeforeEach(func() {
					stackStatus = string(cloudformationtypes.StackStatusRollbackComplete)
					mockIamAPI.EXPECT().GetUser(context.Background(), gomock.Any()).Return(
						&iam.GetUserOutput{User: &iamtypes.User{UserName: &adminUserName}},
						&iamtypes.NoSuchEntityException{},
					)
				})

				It("Returns error telling the stack is in an invalid state", func() {
					stackCreated, err := client.EnsureOsdCcsAdminUser(stackName, adminUserName, DefaultRegion)

					Expect(stackCreated).To(BeFalse())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(
						"exists with status ROLLBACK_COMPLETE. Expected status is CREATE_COMPLETE"))
				})
			})
		})

		Context("When the cloudformation stack does not exists", func() {
			BeforeEach(func() {
				mockCfAPI.EXPECT().ListStacks(context.Background(), gomock.Any()).Return(&cloudformation.ListStacksOutput{
					StackSummaries: []cloudformationtypes.StackSummary{},
				}, nil)
				mockIamAPI.EXPECT().ListUsers(context.Background(), gomock.Any()).Return(
					&iam.ListUsersOutput{Users: []iamtypes.User{}}, nil)
				mockIamAPI.EXPECT().TagUser(context.Background(), gomock.Any()).Return(&iam.TagUserOutput{}, nil)
				mockIamAPI.EXPECT().GetUser(context.Background(), gomock.Any()).Return(
					&iam.GetUserOutput{User: &iamtypes.User{UserName: &adminUserName}},
					&iamtypes.NoSuchEntityException{},
				)
				describeStacksOutput := &cloudformation.DescribeStacksOutput{
					Stacks: []cloudformationtypes.Stack{
						{
							StackName:   &stackName,
							StackStatus: cloudformationtypes.StackStatusCreateComplete,
						},
					},
				}
				mockCfAPI.EXPECT().
					DescribeStacks(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ *cloudformation.DescribeStacksInput,
						_ ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
						return describeStacksOutput, nil
					}).AnyTimes()
				mockCfAPI.EXPECT().CreateStack(context.Background(), gomock.Any()).Return(nil, nil)
			})

			It("Creates a cloudformation stack", func() {
				stackCreated, err := client.EnsureOsdCcsAdminUser(stackName, adminUserName, DefaultRegion)

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
			mockIamAPI.EXPECT().ListUsers(context.Background(), gomock.Any()).Return(&iam.ListUsersOutput{
				Users: []iamtypes.User{
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
		var tags = []iamtypes.Tag{
			{Key: awsSdk.String(common.ManagedPolicies), Value: awsSdk.String(rosaTags.True)},
			{Key: awsSdk.String(rosaTags.RoleType), Value: awsSdk.String(InstallerAccountRole)},
		}

		It("Finds and Returns Account Role", func() {

			mockIamAPI.EXPECT().GetRole(context.Background(), gomock.Any()).Return(&iam.GetRoleOutput{
				Role: &iamtypes.Role{
					Arn:      &testArn,
					RoleName: &testName,
				},
			}, nil)

			mockIamAPI.EXPECT().ListRoleTags(context.Background(), gomock.Any()).Return(&iam.ListRoleTagsOutput{
				Tags: tags,
			}, nil)

			role, err := client.GetAccountRoleByArn(testArn)

			Expect(err).NotTo(HaveOccurred())
			Expect(role).NotTo(BeNil())

			Expect(role.RoleName).To(Equal(testName))
			Expect(role.RoleARN).To(Equal(testArn))
			Expect(role.RoleType).To(Equal(InstallerAccountRoleType))
		})

		It("Returns empty when No Role with ARN exists", func() {
			mockIamAPI.EXPECT().GetRole(context.Background(), gomock.Any()).Return(nil, fmt.Errorf("role Doesn't Exist"))

			role, err := client.GetAccountRoleByArn(testArn)

			Expect(role).To(BeZero())
			Expect(err).To(HaveOccurred())
		})

		It("Returns empty when the Role exists, but it is not an Account Role", func() {

			var roleName = "not-an-account-role"

			mockIamAPI.EXPECT().GetRole(context.Background(), gomock.Any()).Return(&iam.GetRoleOutput{
				Role: &iamtypes.Role{
					Arn:      &testArn,
					RoleName: &roleName,
				},
			}, nil)

			role, err := client.GetAccountRoleByArn(testArn)
			Expect(role).To(BeZero())
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("List Subnets", func() {

		subnetOneId := "test-subnet-1"
		subnetTwoId := "test-subnet-2"
		subnet := ec2types.Subnet{
			SubnetId: helpers.NewString(subnetOneId),
		}

		subnet2 := ec2types.Subnet{
			SubnetId: helpers.NewString(subnetTwoId),
		}

		var subnets []ec2types.Subnet
		subnets = append(subnets, subnet, subnet2)

		It("Lists all", func() {

			var request *ec2.DescribeSubnetsInput

			mockEC2API.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, params *ec2.DescribeSubnetsInput,
					optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
					request = params
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

			mockEC2API.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, params *ec2.DescribeSubnetsInput,
					optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
					request = params
					return &ec2.DescribeSubnetsOutput{
						Subnets: subnets,
					}, nil
				})

			subs, err := client.ListSubnets(subnetOneId, subnetTwoId)
			Expect(err).NotTo(HaveOccurred())

			Expect(subs).To(HaveLen(2))
			Expect(request.SubnetIds).To(ContainElements(subnetOneId, subnetTwoId))

		})
	})

	Context("FetchPublicSubnetMap", func() {

		subnetOneId := "test-subnet-1"
		subnetTwoId := "test-subnet-2"
		subnet := ec2types.Subnet{
			SubnetId: helpers.NewString(subnetOneId),
		}

		subnet2 := ec2types.Subnet{
			SubnetId: helpers.NewString(subnetTwoId),
		}

		var subnets []ec2types.Subnet
		subnets = append(subnets, subnet, subnet2)

		It("Fetches", func() {
			subnetIds := []*string{}
			for _, subnet := range subnets {
				subnetIds = append(subnetIds, subnet.SubnetId)
			}
			input := &ec2.DescribeRouteTablesInput{
				Filters: []ec2types.Filter{
					{
						Name:   awsSdk.String("association.subnet-id"),
						Values: awsSdk.ToStringSlice(subnetIds),
					},
				},
			}
			output := &ec2.DescribeRouteTablesOutput{
				RouteTables: []ec2types.RouteTable{
					{
						Associations: []ec2types.RouteTableAssociation{
							{
								SubnetId: awsSdk.String(subnetOneId),
							},
						},
						Routes: []ec2types.Route{
							{
								GatewayId: awsSdk.String("igw-test"),
							},
						},
					},
					{
						Associations: []ec2types.RouteTableAssociation{
							{
								SubnetId: awsSdk.String(subnetTwoId),
							},
						},
						Routes: []ec2types.Route{
							{
								GatewayId: awsSdk.String("test"),
							},
						},
					},
				},
			}
			mockEC2API.EXPECT().DescribeRouteTables(gomock.Any(), input).Return(output, nil)

			publicSubnetMap, err := client.FetchPublicSubnetMap(subnets)
			Expect(err).NotTo(HaveOccurred())

			Expect(publicSubnetMap).To(HaveLen(2))
			mapStr := fmt.Sprintf("%v", publicSubnetMap)
			Expect(mapStr).To(ContainSubstring("test-subnet-1:true"))
			Expect(mapStr).To(ContainSubstring("test-subnet-2:false"))
		})
	})

	Context("ValidateCredentials", func() {

		It("Wraps InvalidClientTokenId to get user login information", func() {

			err := fmt.Errorf("InvalidClientTokenId: bad credentials")
			mockSTSApi.EXPECT().GetCallerIdentity(&sts.GetCallerIdentityInput{}).Return(nil, err)

			valid, err := client.ValidateCredentials()
			Expect(valid).To(BeFalse())
			Expect(err.Error()).To(ContainSubstring(
				"Invalid AWS Credentials. For help configuring your credentials, see"))
		})

		It("Does not wrap other errors and returns false", func() {
			fakeError := "Fake AWS creds failure"

			err := fmt.Errorf(fakeError)
			mockSTSApi.EXPECT().GetCallerIdentity(&sts.GetCallerIdentityInput{}).Return(nil, err)

			valid, err := client.ValidateCredentials()
			Expect(valid).To(BeFalse())
			Expect(err.Error()).To(ContainSubstring(fakeError))
		})

		It("Returns true if getCallerIdentity has no errors", func() {
			mockSTSApi.EXPECT().GetCallerIdentity(&sts.GetCallerIdentityInput{}).Return(nil, nil)

			valid, err := client.ValidateCredentials()
			Expect(valid).To(BeTrue())
			Expect(err).To(BeNil())
		})
	})

	Context("ShouldRetry", func() {
		var customRetryer CustomRetryer
		var mockRequest *request.Request
		var mockRequestHeader http.Header
		BeforeEach(func() {
			customRetryer = buildCustomRetryer()
		})
		It("Should not retry with 500 status code", func() {
			mockRequest = &request.Request{
				HTTPResponse: &http.Response{
					StatusCode: 500,
				},
			}
			retry := customRetryer.ShouldRetry(mockRequest)
			Expect(retry).To(BeFalse())
		})
		It("Should retry with non 500 status code", func() {
			mockRequestHeader = http.Header{}
			mockRequest = &request.Request{
				HTTPResponse: &http.Response{
					StatusCode: 429,
				},
				HTTPRequest: &http.Request{
					Header: mockRequestHeader,
					Method: "GET",
					URL: &url.URL{
						Host: "test.com",
					},
				},
				Error: errors.New("Throttling"),
			}
			retry := customRetryer.ShouldRetry(mockRequest)
			Expect(retry).ToNot(BeFalse())
		})
	})
})
