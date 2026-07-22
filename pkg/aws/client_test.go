package aws

import (
	"context"
	"fmt"

	gomock "go.uber.org/mock/gomock"

	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cloudformationtypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	secretsmanagertypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	common "github.com/openshift-online/ocm-common/pkg/aws/validations"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/helpers"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/assets"
	"github.com/openshift/rosa/pkg/aws/mocks"
	rosaTags "github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/test/matchers"
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
		mockSTSApi            *mocks.MockStsApiClient

		mockedOidcProviderArns []string
		mockedOidcConfigs      []*cmv1.OidcConfig
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCfAPI = mocks.NewMockCloudFormationApiClient(mockCtrl)
		mockIamAPI = mocks.NewMockIamApiClient(mockCtrl)
		mockEC2API = mocks.NewMockEc2ApiClient(mockCtrl)
		mockS3API = mocks.NewMockS3ApiClient(mockCtrl)
		mockSTSApi = mocks.NewMockStsApiClient(mockCtrl)
		mockSecretsManagerAPI = mocks.NewMockSecretsManagerApiClient(mockCtrl)
		client = New(
			awsSdk.Config{},
			NewLoggerWrapper(logrus.New(), nil),
			mockIamAPI,
			mockEC2API,
			mocks.NewMockOrganizationsApiClient(mockCtrl),
			mockS3API,
			mockSecretsManagerAPI,
			mockSTSApi,
			mockCfAPI,
			mocks.NewMockServiceQuotasApiClient(mockCtrl),
			mocks.NewMockServiceQuotasApiClient(mockCtrl),
			&AccessKey{},
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
				mockCfAPI.EXPECT().ListStacks(gomock.Any(),
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
					mockIamAPI.EXPECT().GetUser(gomock.Any(),
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
					mockIamAPI.EXPECT().ListUsers(gomock.Any(), gomock.Any()).Return(
						&iam.ListUsersOutput{Users: []iamtypes.User{}}, nil)
					mockIamAPI.EXPECT().TagUser(gomock.Any(), gomock.Any()).Return(&iam.TagUserOutput{}, nil)
					mockIamAPI.EXPECT().GetUser(gomock.Any(), &iam.GetUserInput{UserName: &adminUserName}).Return(
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
					mockCfAPI.EXPECT().CreateStack(gomock.Any(), gomock.Any()).Return(nil, nil)
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
					mockIamAPI.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(
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
				mockCfAPI.EXPECT().ListStacks(gomock.Any(), gomock.Any()).Return(&cloudformation.ListStacksOutput{
					StackSummaries: []cloudformationtypes.StackSummary{},
				}, nil)
				mockIamAPI.EXPECT().ListUsers(gomock.Any(), gomock.Any()).Return(
					&iam.ListUsersOutput{Users: []iamtypes.User{}}, nil)
				mockIamAPI.EXPECT().TagUser(gomock.Any(), gomock.Any()).Return(&iam.TagUserOutput{}, nil)
				mockIamAPI.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(
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
				Expect(err.Error()).To(ContainSubstring("error creating user: IAM user"))
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

	Context("when DescribeSecurityGroups is successful", func() {
		var (
			vpcId         string
			securityGroup ec2types.SecurityGroup
		)
		BeforeEach(func() {
			vpcId = "vpc-123456"
			securityGroup = ec2types.SecurityGroup{
				GroupId:   awsSdk.String("sg-123456"),
				GroupName: awsSdk.String("test-group"),
				Tags: []ec2types.Tag{
					{
						Key:   awsSdk.String("Name"),
						Value: awsSdk.String("test-value"),
					},
				},
			}
		})
		It("should return a list of security group IDs", func() {
			mockEC2API.EXPECT().DescribeSecurityGroups(gomock.Any(), gomock.Any()).Return(
				&ec2.DescribeSecurityGroupsOutput{
					SecurityGroups: []ec2types.SecurityGroup{securityGroup},
					NextToken:      nil,
				}, nil,
			)

			securityGroups, err := client.GetSecurityGroupIds(vpcId)
			Expect(err).NotTo(HaveOccurred())
			Expect(securityGroups).To(HaveLen(1))
			Expect(securityGroups[0].GroupId).To(Equal(awsSdk.String("sg-123456")))
		})
	})

	Context("GetCallerIdentity", func() {
		It("Gets caller identity with no error", func() {
			mockSTSApi.EXPECT().GetCallerIdentity(gomock.Any(), &sts.GetCallerIdentityInput{}).
				Return(&sts.GetCallerIdentityOutput{}, nil)

			out, err := client.GetCallerIdentity()
			Expect(out).To(BeEquivalentTo(&sts.GetCallerIdentityOutput{}))
			Expect(err).NotTo(HaveOccurred())
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

			errMsg := fmt.Errorf("InvalidClientTokenId: bad credentials")
			mockSTSApi.EXPECT().GetCallerIdentity(gomock.Any(), &sts.GetCallerIdentityInput{}).Return(nil, errMsg)

			valid, err := client.ValidateCredentials()
			Expect(valid).To(BeFalse())
			expectedErr := fmt.Sprintf("invalid AWS Credentials: %s.\n For help configuring your credentials, see", errMsg)
			Expect(err.Error()).To(ContainSubstring(expectedErr))
		})

		It("Does not wrap other errors and returns false", func() {
			fakeError := "Fake AWS creds failure"

			err := fmt.Errorf("%s", fakeError)
			mockSTSApi.EXPECT().GetCallerIdentity(gomock.Any(), &sts.GetCallerIdentityInput{}).Return(nil, err)

			valid, err := client.ValidateCredentials()
			Expect(valid).To(BeFalse())
			Expect(err.Error()).To(ContainSubstring(fakeError))
		})

		It("Returns true if getCallerIdentity has no errors", func() {
			mockSTSApi.EXPECT().GetCallerIdentity(gomock.Any(), &sts.GetCallerIdentityInput{}).Return(nil, nil)

			valid, err := client.ValidateCredentials()
			Expect(valid).To(BeTrue())
			Expect(err).To(BeNil())
		})
	})
	Describe("Creator", func() {
		DescribeTable("should be adapted from STS caller identity", func(
			identity *sts.GetCallerIdentityOutput,
			expected *Creator,
			expectedError string,
		) {
			creator, err := CreatorForCallerIdentity(identity)
			if expectedError == "" {
				Expect(err).To(BeNil())
			} else {
				Expect(err).To(MatchError(ContainSubstring(expectedError)))
			}
			Expect(creator).To(matchers.MatchExpected(expected))
		},
			Entry(
				"iam user",
				&sts.GetCallerIdentityOutput{
					Arn: awsSdk.String("arn:aws:iam::123456789012:user/David"),
				},
				&Creator{
					ARN:        "arn:aws:iam::123456789012:user/David",
					AccountID:  "123456789012",
					IsSTS:      false,
					IsGovcloud: false,
					Partition:  "aws",
				},
				"",
			),
			Entry(
				"sts identity",
				&sts.GetCallerIdentityOutput{
					Arn: awsSdk.String("arn:aws:sts::123456789123:assumed-role/OrganizationAccountAccessRole/UserAccess"),
				},
				&Creator{
					ARN:        "arn:aws:iam::123456789123:role/OrganizationAccountAccessRole",
					AccountID:  "123456789123",
					IsSTS:      true,
					IsGovcloud: false,
					Partition:  "aws",
				},
				"",
			),
			Entry(
				"gov cloud iam user",
				&sts.GetCallerIdentityOutput{
					Arn: awsSdk.String("arn:aws-us-gov:iam::123456789012:user/David"),
				},
				&Creator{
					ARN:        "arn:aws-us-gov:iam::123456789012:user/David",
					AccountID:  "123456789012",
					IsSTS:      false,
					IsGovcloud: true,
					Partition:  "aws-us-gov",
				},
				"",
			),
			Entry(
				"gov cloud sts identity",
				&sts.GetCallerIdentityOutput{
					Arn: awsSdk.String("arn:aws-us-gov:sts::123456789123:assumed-role/OrganizationAccountAccessRole/UserAccess"),
				},
				&Creator{
					ARN:        "arn:aws-us-gov:iam::123456789123:role/OrganizationAccountAccessRole",
					AccountID:  "123456789123",
					IsSTS:      true,
					IsGovcloud: true,
					Partition:  "aws-us-gov",
				},
				"",
			),
		)
	})

	Describe("oidc-provider", func() {
		Context("Filtering test", func() {
			BeforeEach(func() {
				mockedOidcProviderArns = []string{
					"arn:aws:iam::765374464689:oidc-provider/oidc.test1/123123123123",
					"arn:aws:iam::765374464689:oidc-provider/oidc.test2/234234234234",
					"arn:aws:iam::765374464689:oidc-provider/oidc.test3/345345345345",
					"arn:aws:iam::765374464689:oidc-provider/oidc.test123/456456456456",
				}
				config1, err := MockOidcConfig("config1", "http://oidc.test1/123123123123")
				Expect(err).ToNot(HaveOccurred())
				config2, err := MockOidcConfig("config2", "http://oidc.test2/234234234234")
				Expect(err).ToNot(HaveOccurred())
				config3, err := MockOidcConfig("config1", "http://oidc.test3/345345345345")
				Expect(err).ToNot(HaveOccurred())
				mockedOidcConfigs = []*cmv1.OidcConfig{config1, config2, config3}
				mockIamAPI.EXPECT().ListOpenIDConnectProviders(gomock.Any(), gomock.Any()).Return(&iam.
					ListOpenIDConnectProvidersOutput{OpenIDConnectProviderList: []iamtypes.OpenIDConnectProviderListEntry{
					{Arn: awsSdk.String(mockedOidcProviderArns[0])},
					{Arn: awsSdk.String(mockedOidcProviderArns[1])},
					{Arn: awsSdk.String(mockedOidcProviderArns[2])},
					{Arn: awsSdk.String(mockedOidcProviderArns[3])},
				}}, nil)
				mockIamAPI.EXPECT().ListOpenIDConnectProviderTags(gomock.Any(), gomock.Any()).Return(
					&iam.ListOpenIDConnectProviderTagsOutput{IsTruncated: *awsSdk.Bool(false),
						Marker: awsSdk.String(""), Tags: []iamtypes.Tag{{Key: awsSdk.String(rosaTags.RedHatManaged),
							Value: awsSdk.String("true")}, {Key: awsSdk.String(rosaTags.ClusterID),
							Value: awsSdk.String("")}}}, nil).AnyTimes()
			})
			It("Filters config 1's ID (1 provider)", func() {
				providers, err := client.ListOidcProviders("", mockedOidcConfigs[0])
				Expect(err).NotTo(HaveOccurred())
				Expect(len(providers)).To(Equal(1))
				Expect(providers[0].Arn).To(Equal(mockedOidcProviderArns[0]))
			})
			It("Filters config 2'd ID (1 provider)", func() {
				providers, err := client.ListOidcProviders("", mockedOidcConfigs[1])
				Expect(err).NotTo(HaveOccurred())
				Expect(len(providers)).To(Equal(1))
				Expect(providers[0].Arn).To(Equal(mockedOidcProviderArns[1]))
			})
			It("Filters config 3's ID (1 provider)", func() {
				providers, err := client.ListOidcProviders("", mockedOidcConfigs[2])
				Expect(err).NotTo(HaveOccurred())
				Expect(len(providers)).To(Equal(1))
				Expect(providers[0].Arn).To(Equal(mockedOidcProviderArns[2]))
			})
			It("Filters nothing (all providers return)", func() {
				providers, err := client.ListOidcProviders("", nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(providers)).To(Equal(4))
			})
		})
	})

	Context("AvailabilityZoneType", func() {

		zoneName := "us-east-1a"

		It("Fetches", func() {
			input := &ec2.DescribeAvailabilityZonesInput{
				ZoneNames: []string{zoneName},
			}
			output := &ec2.DescribeAvailabilityZonesOutput{
				AvailabilityZones: []ec2types.AvailabilityZone{
					{
						ZoneType: awsSdk.String(LocalZone),
					},
				},
			}

			mockEC2API.EXPECT().DescribeAvailabilityZones(gomock.Any(), input).Return(output, nil)

			zoneType, err := client.GetAvailabilityZoneType(zoneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(zoneType).To(Equal(LocalZone))
		})
	})

	Describe("Creator", func() {
		DescribeTable("should be adapted from STS caller identity", func(
			identity *sts.GetCallerIdentityOutput,
			expected *Creator,
			expectedError string,
		) {
			creator, err := CreatorForCallerIdentity(identity)
			if expectedError == "" {
				Expect(err).To(BeNil())
			} else {
				Expect(err).To(MatchError(ContainSubstring(expectedError)))
			}
			Expect(creator).To(matchers.MatchExpected(expected))
		},
			Entry(
				"iam user",
				&sts.GetCallerIdentityOutput{
					Arn: awsSdk.String("arn:aws:iam::123456789012:user/David"),
				},
				&Creator{
					ARN:        "arn:aws:iam::123456789012:user/David",
					AccountID:  "123456789012",
					IsSTS:      false,
					IsGovcloud: false,
					Partition:  "aws",
				},
				"",
			),
			Entry(
				"sts identity",
				&sts.GetCallerIdentityOutput{
					Arn: awsSdk.String("arn:aws:sts::123456789123:assumed-role/OrganizationAccountAccessRole/UserAccess"),
				},
				&Creator{
					ARN:        "arn:aws:iam::123456789123:role/OrganizationAccountAccessRole",
					AccountID:  "123456789123",
					IsSTS:      true,
					IsGovcloud: false,
					Partition:  "aws",
				},
				"",
			),
			Entry(
				"gov cloud iam user",
				&sts.GetCallerIdentityOutput{
					Arn: awsSdk.String("arn:aws-us-gov:iam::123456789012:user/David"),
				},
				&Creator{
					ARN:        "arn:aws-us-gov:iam::123456789012:user/David",
					AccountID:  "123456789012",
					IsSTS:      false,
					IsGovcloud: true,
					Partition:  "aws-us-gov",
				},
				"",
			),
			Entry(
				"gov cloud sts identity",
				&sts.GetCallerIdentityOutput{
					Arn: awsSdk.String("arn:aws-us-gov:sts::123456789123:assumed-role/OrganizationAccountAccessRole/UserAccess"),
				},
				&Creator{
					ARN:        "arn:aws-us-gov:iam::123456789123:role/OrganizationAccountAccessRole",
					AccountID:  "123456789123",
					IsSTS:      true,
					IsGovcloud: true,
					Partition:  "aws-us-gov",
				},
				"",
			),
		)
	})

	Context("ListServiceAccountRoles", func() {
		var (
			clusterName string
		)

		BeforeEach(func() {
			clusterName = "test-cluster"
		})

		It("should list roles with proper tag filtering", func() {
			// Mock ListRoles response
			role1 := "test-cluster-ns1-app1-role"
			role2 := "test-cluster-ns2-app2-role"
			role3 := "other-role-without-tags"

			mockIamAPI.EXPECT().ListRoles(gomock.Any(), gomock.Any(), gomock.Any()).Return(&iam.ListRolesOutput{
				Roles: []iamtypes.Role{
					{
						RoleName: &role1,
						Arn:      awsSdk.String("arn:aws:iam::123456789012:role/" + role1),
					},
					{
						RoleName: &role2,
						Arn:      awsSdk.String("arn:aws:iam::123456789012:role/" + role2),
					},
					{
						RoleName: &role3,
						Arn:      awsSdk.String("arn:aws:iam::123456789012:role/" + role3),
					},
				},
			}, nil)

			// Mock ListRoleTags for each role
			mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), &iam.ListRoleTagsInput{
				RoleName: &role1,
			}).Return(&iam.ListRoleTagsOutput{
				Tags: []iamtypes.Tag{
					{Key: awsSdk.String("rosa_role_type"), Value: awsSdk.String("ServiceAccountRole")},
					{Key: awsSdk.String("rosa.openshift.io/cluster"), Value: awsSdk.String("test-cluster")},
					{Key: awsSdk.String("rosa.openshift.io/namespace"), Value: awsSdk.String("ns1")},
					{Key: awsSdk.String("rosa.openshift.io/service-account"), Value: awsSdk.String("app1")},
				},
			}, nil)

			mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), &iam.ListRoleTagsInput{
				RoleName: &role2,
			}).Return(&iam.ListRoleTagsOutput{
				Tags: []iamtypes.Tag{
					{Key: awsSdk.String("rosa_role_type"), Value: awsSdk.String("ServiceAccountRole")},
					{Key: awsSdk.String("rosa.openshift.io/cluster"), Value: awsSdk.String("test-cluster")},
					{Key: awsSdk.String("rosa.openshift.io/namespace"), Value: awsSdk.String("ns2")},
					{Key: awsSdk.String("rosa.openshift.io/service-account"), Value: awsSdk.String("app2")},
				},
			}, nil)

			mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), &iam.ListRoleTagsInput{
				RoleName: &role3,
			}).Return(&iam.ListRoleTagsOutput{
				Tags: []iamtypes.Tag{
					{Key: awsSdk.String("some-other-tag"), Value: awsSdk.String("value")},
				},
			}, nil)

			// Call the function
			roles, err := client.ListServiceAccountRoles(clusterName)

			// Verify results
			Expect(err).ToNot(HaveOccurred())
			Expect(roles).To(HaveLen(2))
		})

		It("should return multiple roles for the same cluster", func() {
			// Mock ListRoles response
			role1 := "test-cluster-ns1-app1-role"
			role2 := "test-cluster-ns1-app2-role"

			mockIamAPI.EXPECT().ListRoles(gomock.Any(), gomock.Any(), gomock.Any()).Return(&iam.ListRolesOutput{
				Roles: []iamtypes.Role{
					{
						RoleName: &role1,
						Arn:      awsSdk.String("arn:aws:iam::123456789012:role/" + role1),
					},
					{
						RoleName: &role2,
						Arn:      awsSdk.String("arn:aws:iam::123456789012:role/" + role2),
					},
				},
			}, nil)

			// Mock ListRoleTags for each role
			mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), &iam.ListRoleTagsInput{
				RoleName: &role1,
			}).Return(&iam.ListRoleTagsOutput{
				Tags: []iamtypes.Tag{
					{Key: awsSdk.String("rosa_role_type"), Value: awsSdk.String("ServiceAccountRole")},
					{Key: awsSdk.String("rosa.openshift.io/cluster"), Value: awsSdk.String("test-cluster")},
					{Key: awsSdk.String("rosa.openshift.io/namespace"), Value: awsSdk.String("ns1")},
					{Key: awsSdk.String("rosa.openshift.io/service-account"), Value: awsSdk.String("app1")},
				},
			}, nil)

			mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), &iam.ListRoleTagsInput{
				RoleName: &role2,
			}).Return(&iam.ListRoleTagsOutput{
				Tags: []iamtypes.Tag{
					{Key: awsSdk.String("rosa_role_type"), Value: awsSdk.String("ServiceAccountRole")},
					{Key: awsSdk.String("rosa.openshift.io/cluster"), Value: awsSdk.String("test-cluster")},
					{Key: awsSdk.String("rosa.openshift.io/namespace"), Value: awsSdk.String("ns1")},
					{Key: awsSdk.String("rosa.openshift.io/service-account"), Value: awsSdk.String("app2")},
				},
			}, nil)

			// Call the function
			roles, err := client.ListServiceAccountRoles(clusterName)

			// Verify results
			Expect(err).ToNot(HaveOccurred())
			Expect(roles).To(HaveLen(2))
			Expect(*roles[0].RoleName).To(Equal(role1))
			Expect(*roles[1].RoleName).To(Equal(role2))
		})

		It("should handle ListRoles errors", func() {
			mockIamAPI.EXPECT().ListRoles(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("aws error"))

			roles, err := client.ListServiceAccountRoles(clusterName)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("aws error"))
			Expect(roles).To(BeNil())
		})

		It("should handle ListRoleTags errors gracefully", func() {
			// Mock ListRoles response
			role1 := "test-cluster-ns1-app1-role"

			mockIamAPI.EXPECT().ListRoles(gomock.Any(), gomock.Any(), gomock.Any()).Return(&iam.ListRolesOutput{
				Roles: []iamtypes.Role{
					{
						RoleName: &role1,
						Arn:      awsSdk.String("arn:aws:iam::123456789012:role/" + role1),
					},
				},
			}, nil)

			// Mock ListRoleTags to return an error
			mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), &iam.ListRoleTagsInput{
				RoleName: &role1,
			}).Return(nil, fmt.Errorf("tag error"))

			// Call the function
			roles, err := client.ListServiceAccountRoles(clusterName)

			// Should succeed but with empty results (error is logged but not returned)
			Expect(err).ToNot(HaveOccurred())
			Expect(roles).To(HaveLen(0))
		})

		It("should handle roles without required tags", func() {
			// Mock ListRoles response
			role1 := "test-cluster-ns1-app1-role"

			mockIamAPI.EXPECT().ListRoles(gomock.Any(), gomock.Any(), gomock.Any()).Return(&iam.ListRolesOutput{
				Roles: []iamtypes.Role{
					{
						RoleName: &role1,
						Arn:      awsSdk.String("arn:aws:iam::123456789012:role/" + role1),
					},
				},
			}, nil)

			// Mock ListRoleTags with missing required tags
			mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), &iam.ListRoleTagsInput{
				RoleName: &role1,
			}).Return(&iam.ListRoleTagsOutput{
				Tags: []iamtypes.Tag{
					{Key: awsSdk.String("rosa_role_type"), Value: awsSdk.String("ServiceAccountRole")},
					// Missing cluster tag
					{Key: awsSdk.String("rosa.openshift.io/namespace"), Value: awsSdk.String("ns1")},
				},
			}, nil)

			// Call the function
			roles, err := client.ListServiceAccountRoles(clusterName)

			// Should succeed but with empty results (role doesn't match cluster)
			Expect(err).ToNot(HaveOccurred())
			Expect(roles).To(HaveLen(0))
		})
	})

	Context("CreateS3Bucket", func() {
		bucketName := "test-oidc-bucket"

		It("Returns error when bucket already exists", func() {
			mockS3API.EXPECT().HeadBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.HeadBucketOutput{}, nil)

			err := client.CreateS3Bucket(bucketName, DefaultRegion)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})

		It("Creates bucket in default region without LocationConstraint", func() {
			mockS3API.EXPECT().HeadBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("not found"))

			mockS3API.EXPECT().CreateBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, input *s3.CreateBucketInput,
					_ ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
					Expect(*input.Bucket).To(Equal(bucketName))
					Expect(input.CreateBucketConfiguration).To(BeNil())
					return &s3.CreateBucketOutput{}, nil
				})

			mockS3API.EXPECT().PutPublicAccessBlock(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.PutPublicAccessBlockOutput{}, nil)

			mockS3API.EXPECT().PutBucketPolicy(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.PutBucketPolicyOutput{}, nil)

			mockS3API.EXPECT().PutBucketTagging(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.PutBucketTaggingOutput{}, nil)

			err := client.CreateS3Bucket(bucketName, DefaultRegion)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Creates bucket in non-default region with LocationConstraint", func() {
			nonDefaultRegion := "eu-west-1"

			mockS3API.EXPECT().HeadBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("not found"))

			mockS3API.EXPECT().CreateBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, input *s3.CreateBucketInput,
					_ ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
					Expect(*input.Bucket).To(Equal(bucketName))
					Expect(input.CreateBucketConfiguration).NotTo(BeNil())
					Expect(input.CreateBucketConfiguration.LocationConstraint).To(
						Equal(s3types.BucketLocationConstraint(nonDefaultRegion)))
					return &s3.CreateBucketOutput{}, nil
				})

			mockS3API.EXPECT().PutPublicAccessBlock(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.PutPublicAccessBlockOutput{}, nil)

			mockS3API.EXPECT().PutBucketPolicy(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.PutBucketPolicyOutput{}, nil)

			mockS3API.EXPECT().PutBucketTagging(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.PutBucketTaggingOutput{}, nil)

			err := client.CreateS3Bucket(bucketName, nonDefaultRegion)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Returns error when CreateBucket fails", func() {
			mockS3API.EXPECT().HeadBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("not found"))

			mockS3API.EXPECT().CreateBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("bucket creation failed"))

			err := client.CreateS3Bucket(bucketName, DefaultRegion)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bucket creation failed"))
		})

		It("Returns error when PutPublicAccessBlock fails", func() {
			mockS3API.EXPECT().HeadBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("not found"))

			mockS3API.EXPECT().CreateBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.CreateBucketOutput{}, nil)

			mockS3API.EXPECT().PutPublicAccessBlock(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("access block error"))

			err := client.CreateS3Bucket(bucketName, DefaultRegion)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("access block error"))
		})

		It("Returns error when PutBucketPolicy fails", func() {
			mockS3API.EXPECT().HeadBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("not found"))

			mockS3API.EXPECT().CreateBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.CreateBucketOutput{}, nil)

			mockS3API.EXPECT().PutPublicAccessBlock(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.PutPublicAccessBlockOutput{}, nil)

			mockS3API.EXPECT().PutBucketPolicy(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("policy error"))

			err := client.CreateS3Bucket(bucketName, DefaultRegion)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("policy error"))
		})

		It("Returns error when PutBucketTagging fails", func() {
			mockS3API.EXPECT().HeadBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("not found"))

			mockS3API.EXPECT().CreateBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.CreateBucketOutput{}, nil)

			mockS3API.EXPECT().PutPublicAccessBlock(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.PutPublicAccessBlockOutput{}, nil)

			mockS3API.EXPECT().PutBucketPolicy(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.PutBucketPolicyOutput{}, nil)

			mockS3API.EXPECT().PutBucketTagging(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("tagging error"))

			err := client.CreateS3Bucket(bucketName, DefaultRegion)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("tagging error"))
		})
	})

	Context("DeleteS3Bucket", func() {
		bucketName := "test-oidc-bucket"

		It("Returns nil when bucket is not found", func() {
			mockS3API.EXPECT().HeadBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, &s3types.NotFound{})

			err := client.DeleteS3Bucket(bucketName)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Empties bucket and deletes it successfully", func() {
			mockS3API.EXPECT().HeadBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.HeadBucketOutput{}, nil)

			mockS3API.EXPECT().ListObjects(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.ListObjectsOutput{
					Contents: []s3types.Object{
						{Key: awsSdk.String("file1.json")},
						{Key: awsSdk.String("file2.json")},
					},
				}, nil)

			mockS3API.EXPECT().DeleteObject(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.DeleteObjectOutput{}, nil).Times(2)

			mockS3API.EXPECT().DeleteBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.DeleteBucketOutput{}, nil)

			err := client.DeleteS3Bucket(bucketName)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Returns error when ListObjects fails", func() {
			mockS3API.EXPECT().HeadBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.HeadBucketOutput{}, nil)

			mockS3API.EXPECT().ListObjects(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("list objects error"))

			err := client.DeleteS3Bucket(bucketName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("list objects error"))
		})

		It("Returns error when DeleteBucket fails", func() {
			mockS3API.EXPECT().HeadBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.HeadBucketOutput{}, nil)

			mockS3API.EXPECT().ListObjects(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&s3.ListObjectsOutput{Contents: []s3types.Object{}}, nil)

			mockS3API.EXPECT().DeleteBucket(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("delete bucket error"))

			err := client.DeleteS3Bucket(bucketName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("delete bucket error"))
		})
	})

	Context("CreateSecretInSecretsManager", func() {
		It("Returns ARN on success", func() {
			expectedArn := "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-abc123"

			mockSecretsManagerAPI.EXPECT().CreateSecret(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, input *secretsmanager.CreateSecretInput,
					_ ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
					Expect(*input.Name).To(Equal("my-secret"))
					Expect(*input.SecretString).To(Equal("secret-value"))
					return &secretsmanager.CreateSecretOutput{
						ARN: awsSdk.String(expectedArn),
					}, nil
				})

			arn, err := client.CreateSecretInSecretsManager("my-secret", "secret-value")
			Expect(err).NotTo(HaveOccurred())
			Expect(arn).To(Equal(expectedArn))
		})

		It("Returns error when CreateSecret fails", func() {
			mockSecretsManagerAPI.EXPECT().CreateSecret(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("permission denied"))

			arn, err := client.CreateSecretInSecretsManager("my-secret", "secret-value")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("permission denied"))
			Expect(arn).To(BeEmpty())
		})
	})

	Context("DeleteSecretInSecretsManager", func() {
		secretArn := "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-abc123"

		It("Returns nil when secret is not found", func() {
			mockSecretsManagerAPI.EXPECT().DescribeSecret(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, &secretsmanagertypes.ResourceNotFoundException{
					Message: awsSdk.String("not found"),
				})

			err := client.DeleteSecretInSecretsManager(secretArn)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Deletes secret successfully when it exists", func() {
			mockSecretsManagerAPI.EXPECT().DescribeSecret(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, input *secretsmanager.DescribeSecretInput,
					_ ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error) {
					Expect(*input.SecretId).To(Equal(secretArn))
					return &secretsmanager.DescribeSecretOutput{
						ARN:  awsSdk.String(secretArn),
						Name: awsSdk.String("my-secret"),
					}, nil
				})

			mockSecretsManagerAPI.EXPECT().DeleteSecret(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, input *secretsmanager.DeleteSecretInput,
					_ ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error) {
					Expect(*input.SecretId).To(Equal(secretArn))
					Expect(*input.ForceDeleteWithoutRecovery).To(BeTrue())
					return &secretsmanager.DeleteSecretOutput{}, nil
				})

			err := client.DeleteSecretInSecretsManager(secretArn)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Returns error when DeleteSecret fails", func() {
			mockSecretsManagerAPI.EXPECT().DescribeSecret(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&secretsmanager.DescribeSecretOutput{
					ARN: awsSdk.String(secretArn),
				}, nil)

			mockSecretsManagerAPI.EXPECT().DeleteSecret(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("internal service error"))

			err := client.DeleteSecretInSecretsManager(secretArn)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("internal service error"))
		})
	})

	Context("DescribeAvailabilityZones", func() {
		It("Returns zone names on success", func() {
			mockEC2API.EXPECT().DescribeAvailabilityZones(gomock.Any(), gomock.Any()).
				Return(&ec2.DescribeAvailabilityZonesOutput{
					AvailabilityZones: []ec2types.AvailabilityZone{
						{ZoneName: awsSdk.String("us-east-1a")},
						{ZoneName: awsSdk.String("us-east-1b")},
						{ZoneName: awsSdk.String("us-east-1c")},
					},
				}, nil)

			zones, err := client.DescribeAvailabilityZones()
			Expect(err).NotTo(HaveOccurred())
			Expect(zones).To(HaveLen(3))
			Expect(zones).To(ContainElements("us-east-1a", "us-east-1b", "us-east-1c"))
		})

		It("Propagates API error", func() {
			mockEC2API.EXPECT().DescribeAvailabilityZones(gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("ec2 error"))

			zones, err := client.DescribeAvailabilityZones()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ec2 error"))
			Expect(zones).To(BeNil())
		})
	})

	Context("IsLocalAvailabilityZone", func() {
		It("Returns true for a local zone", func() {
			mockEC2API.EXPECT().DescribeAvailabilityZones(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, input *ec2.DescribeAvailabilityZonesInput,
					_ ...func(*ec2.Options)) (*ec2.DescribeAvailabilityZonesOutput, error) {
					Expect(input.ZoneNames).To(ContainElement("us-east-1-bos-1a"))
					return &ec2.DescribeAvailabilityZonesOutput{
						AvailabilityZones: []ec2types.AvailabilityZone{
							{ZoneType: awsSdk.String(LocalZone)},
						},
					}, nil
				})

			isLocal, err := client.IsLocalAvailabilityZone("us-east-1-bos-1a")
			Expect(err).NotTo(HaveOccurred())
			Expect(isLocal).To(BeTrue())
		})

		It("Returns false for a non-local zone", func() {
			mockEC2API.EXPECT().DescribeAvailabilityZones(gomock.Any(), gomock.Any()).
				Return(&ec2.DescribeAvailabilityZonesOutput{
					AvailabilityZones: []ec2types.AvailabilityZone{
						{ZoneType: awsSdk.String("availability-zone")},
					},
				}, nil)

			isLocal, err := client.IsLocalAvailabilityZone("us-east-1a")
			Expect(err).NotTo(HaveOccurred())
			Expect(isLocal).To(BeFalse())
		})

		It("Returns error when zone is not found", func() {
			mockEC2API.EXPECT().DescribeAvailabilityZones(gomock.Any(), gomock.Any()).
				Return(&ec2.DescribeAvailabilityZonesOutput{
					AvailabilityZones: []ec2types.AvailabilityZone{},
				}, nil)

			_, err := client.IsLocalAvailabilityZone("nonexistent-zone")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to find availability zone"))
		})

		It("Propagates API error", func() {
			mockEC2API.EXPECT().DescribeAvailabilityZones(gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("api failure"))

			_, err := client.IsLocalAvailabilityZone("us-east-1a")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("api failure"))
		})
	})

	Context("GetSubnetAvailabilityZone", func() {
		It("Returns availability zone on success", func() {
			subnetID := "subnet-12345"
			mockEC2API.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, input *ec2.DescribeSubnetsInput,
					_ ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
					Expect(input.SubnetIds).To(ContainElement(subnetID))
					return &ec2.DescribeSubnetsOutput{
						Subnets: []ec2types.Subnet{
							{
								SubnetId:         awsSdk.String(subnetID),
								AvailabilityZone: awsSdk.String("us-east-1a"),
							},
						},
					}, nil
				})

			az, err := client.GetSubnetAvailabilityZone(subnetID)
			Expect(err).NotTo(HaveOccurred())
			Expect(az).To(Equal("us-east-1a"))
		})

		It("Propagates API error", func() {
			mockEC2API.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("subnet error"))

			_, err := client.GetSubnetAvailabilityZone("subnet-12345")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("subnet error"))
		})
	})

	Context("GetVPCSubnets", func() {
		It("Returns filtered subnets for a VPC", func() {
			subnetID := "subnet-aaa"
			vpcID := "vpc-12345"
			mockEC2API.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, input *ec2.DescribeSubnetsInput,
					_ ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
					for _, f := range input.Filters {
						if *f.Name == "subnet-id" {
							return &ec2.DescribeSubnetsOutput{
								Subnets: []ec2types.Subnet{
									{
										SubnetId: awsSdk.String(subnetID),
										VpcId:    awsSdk.String(vpcID),
									},
								},
							}, nil
						}
					}
					return &ec2.DescribeSubnetsOutput{
						Subnets: []ec2types.Subnet{
							{SubnetId: awsSdk.String(subnetID), VpcId: awsSdk.String(vpcID)},
							{SubnetId: awsSdk.String("subnet-bbb"), VpcId: awsSdk.String(vpcID)},
							{SubnetId: awsSdk.String("subnet-ccc"), VpcId: awsSdk.String(vpcID)},
						},
					}, nil
				}).Times(2)

			subnets, err := client.GetVPCSubnets(subnetID)
			Expect(err).NotTo(HaveOccurred())
			Expect(subnets).To(HaveLen(3))
		})

		It("Propagates error from first DescribeSubnets call", func() {
			mockEC2API.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("describe subnets error"))

			subnets, err := client.GetVPCSubnets("subnet-aaa")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("describe subnets error"))
			Expect(subnets).To(BeNil())
		})
	})

	Context("CheckRoleExists", func() {
		When("the role exists", func() {
			It("returns true and the ARN", func() {
				mockIamAPI.EXPECT().GetRole(gomock.Any(), gomock.Any()).Return(
					&iam.GetRoleOutput{
						Role: &iamtypes.Role{
							RoleName: awsSdk.String("my-role"),
							Arn:      awsSdk.String("arn:aws:iam::123456789012:role/my-role"),
						},
					}, nil)

				exists, roleArn, err := client.CheckRoleExists("my-role")
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
				Expect(roleArn).To(Equal("arn:aws:iam::123456789012:role/my-role"))
			})
		})

		When("the role does not exist", func() {
			It("returns false with empty ARN and nil error", func() {
				mockIamAPI.EXPECT().GetRole(gomock.Any(), gomock.Any()).Return(
					nil, &iamtypes.NoSuchEntityException{})

				exists, roleArn, err := client.CheckRoleExists("nonexistent-role")
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeFalse())
				Expect(roleArn).To(BeEmpty())
			})
		})

		When("GetRole returns another error", func() {
			It("propagates the error", func() {
				mockIamAPI.EXPECT().GetRole(gomock.Any(), gomock.Any()).Return(
					nil, fmt.Errorf("access denied"))

				exists, roleArn, err := client.CheckRoleExists("my-role")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("access denied"))
				Expect(exists).To(BeFalse())
				Expect(roleArn).To(BeEmpty())
			})
		})
	})

	Context("GetRoleByARN", func() {
		When("given a valid role ARN", func() {
			It("returns the role", func() {
				testArn := "arn:aws:iam::123456789012:role/my-role"
				mockIamAPI.EXPECT().GetRole(gomock.Any(), gomock.Any()).Return(
					&iam.GetRoleOutput{
						Role: &iamtypes.Role{
							RoleName: awsSdk.String("my-role"),
							Arn:      awsSdk.String(testArn),
						},
					}, nil)

				role, err := client.GetRoleByARN(testArn)
				Expect(err).ToNot(HaveOccurred())
				Expect(awsSdk.ToString(role.RoleName)).To(Equal("my-role"))
				Expect(awsSdk.ToString(role.Arn)).To(Equal(testArn))
			})
		})

		When("given an invalid ARN string", func() {
			It("returns an error", func() {
				role, err := client.GetRoleByARN("not-a-valid-arn")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("valid IAM role ARN"))
				Expect(role).To(Equal(iamtypes.Role{}))
			})
		})

		When("the ARN is not for a role resource", func() {
			It("returns an error", func() {
				role, err := client.GetRoleByARN("arn:aws:iam::123456789012:user/my-user")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("IAM role resource"))
				Expect(role).To(Equal(iamtypes.Role{}))
			})
		})

		When("GetRole fails", func() {
			It("propagates the error", func() {
				mockIamAPI.EXPECT().GetRole(gomock.Any(), gomock.Any()).Return(
					nil, fmt.Errorf("iam service error"))

				role, err := client.GetRoleByARN("arn:aws:iam::123456789012:role/missing-role")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("iam service error"))
				Expect(role).To(Equal(iamtypes.Role{}))
			})
		})
	})

	Context("GetRoleByName", func() {
		When("the role is found", func() {
			It("returns the role", func() {
				mockIamAPI.EXPECT().GetRole(gomock.Any(), gomock.Any()).Return(
					&iam.GetRoleOutput{
						Role: &iamtypes.Role{
							RoleName: awsSdk.String("my-role"),
							Arn:      awsSdk.String("arn:aws:iam::123456789012:role/my-role"),
						},
					}, nil)

				role, err := client.GetRoleByName("my-role")
				Expect(err).ToNot(HaveOccurred())
				Expect(awsSdk.ToString(role.RoleName)).To(Equal("my-role"))
			})
		})

		When("GetRole returns an error", func() {
			It("propagates the error", func() {
				mockIamAPI.EXPECT().GetRole(gomock.Any(), gomock.Any()).Return(
					nil, fmt.Errorf("role not found"))

				role, err := client.GetRoleByName("missing-role")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("role not found"))
				Expect(role).To(Equal(iamtypes.Role{}))
			})
		})
	})
})
