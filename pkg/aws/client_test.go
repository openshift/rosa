package aws

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	awsSdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	common "github.com/openshift-online/ocm-common/pkg/aws/validations"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/helpers"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/aws/mocks"
	rosaTags "github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/test/matchers"
)

var _ = Describe("Client", func() {
	var (
		client   Client
		mockCtrl *gomock.Controller

		mockEC2API            *mocks.MockEC2API
		mockCfAPI             *mocks.MockCloudFormationAPI
		mockIamAPI            *mocks.MockIAMAPI
		mockS3API             *mocks.MockS3API
		mockSecretsManagerAPI *mocks.MockSecretsManagerAPI
		mockSTSApi            *mocks.MockSTSAPI

		mockedOidcProviderArns []string
		mockedOidcConfigs      []*cmv1.OidcConfig
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCfAPI = mocks.NewMockCloudFormationAPI(mockCtrl)
		mockIamAPI = mocks.NewMockIAMAPI(mockCtrl)
		mockEC2API = mocks.NewMockEC2API(mockCtrl)
		mockS3API = mocks.NewMockS3API(mockCtrl)
		mockSecretsManagerAPI = mocks.NewMockSecretsManagerAPI(mockCtrl)
		mockSTSApi = mocks.NewMockSTSAPI(mockCtrl)
		client = New(
			logrus.New(),
			mockIamAPI,
			mockEC2API,
			mocks.NewMockOrganizationsAPI(mockCtrl),
			mockS3API,
			mockSecretsManagerAPI,
			mockSTSApi,
			mockCfAPI,
			mocks.NewMockServiceQuotasAPI(mockCtrl),
			&session.Session{},
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
					stackCreated, err := client.EnsureOsdCcsAdminUser(stackName, adminUserName, DefaultRegion)

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
					stackCreated, err := client.EnsureOsdCcsAdminUser(stackName, adminUserName, DefaultRegion)

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
			{Key: awsSdk.String(common.ManagedPolicies), Value: awsSdk.String(rosaTags.True)},
			{Key: awsSdk.String(rosaTags.RoleType), Value: awsSdk.String(InstallerAccountRole)},
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

			role, err := client.GetAccountRoleByArn(testArn)

			Expect(err).NotTo(HaveOccurred())
			Expect(role).NotTo(BeNil())

			Expect(role.RoleName).To(Equal(testName))
			Expect(role.RoleARN).To(Equal(testArn))
			Expect(role.RoleType).To(Equal(InstallerAccountRoleType))
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

	Context("FetchPublicSubnetMap", func() {

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

		It("Fetches", func() {
			subnetIds := []*string{}
			for _, subnet := range subnets {
				subnetIds = append(subnetIds, subnet.SubnetId)
			}
			input := &ec2.DescribeRouteTablesInput{
				Filters: []*ec2.Filter{
					{
						Name:   awsSdk.String("association.subnet-id"),
						Values: subnetIds,
					},
				},
			}
			output := &ec2.DescribeRouteTablesOutput{
				RouteTables: []*ec2.RouteTable{
					{
						Associations: []*ec2.RouteTableAssociation{
							{
								SubnetId: awsSdk.String(subnetOneId),
							},
						},
						Routes: []*ec2.Route{
							{
								GatewayId: awsSdk.String("igw-test"),
							},
						},
					},
					{
						Associations: []*ec2.RouteTableAssociation{
							{
								SubnetId: awsSdk.String(subnetTwoId),
							},
						},
						Routes: []*ec2.Route{
							{
								GatewayId: awsSdk.String("test"),
							},
						},
					},
				},
			}
			mockEC2API.EXPECT().DescribeRouteTables(input).Return(output, nil)

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
				mockIamAPI.EXPECT().ListOpenIDConnectProviders(gomock.Any()).Return(&iam.
					ListOpenIDConnectProvidersOutput{OpenIDConnectProviderList: []*iam.OpenIDConnectProviderListEntry{
					{Arn: awsSdk.String(mockedOidcProviderArns[0])},
					{Arn: awsSdk.String(mockedOidcProviderArns[1])},
					{Arn: awsSdk.String(mockedOidcProviderArns[2])},
					{Arn: awsSdk.String(mockedOidcProviderArns[3])},
				}}, nil)
				mockIamAPI.EXPECT().ListOpenIDConnectProviderTags(gomock.Any()).Return(
					&iam.ListOpenIDConnectProviderTagsOutput{IsTruncated: awsSdk.Bool(false),
						Marker: awsSdk.String(""), Tags: []*iam.Tag{{Key: awsSdk.String(rosaTags.RedHatManaged),
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
})
