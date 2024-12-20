package aws

import (
	"errors"
	"fmt"

	gomock "go.uber.org/mock/gomock"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	common "github.com/openshift-online/ocm-common/pkg/aws/validations"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/aws/mocks"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/logging"
)

var _ = Describe("ListOperatorRoles", func() {
	var (
		client     awsClient
		mockIamAPI *mocks.MockIamApiClient
		mockCtrl   *gomock.Controller
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockIamAPI = mocks.NewMockIamApiClient(mockCtrl)
		client = awsClient{
			iamClient: mockIamAPI,
		}
	})

	It("Retrieves by target version", func() {
		mockIamAPI.EXPECT().ListRoles(gomock.Any(), gomock.Any()).Return(
			&iam.ListRolesOutput{
				IsTruncated: false,
				Roles: []iamtypes.Role{
					{
						RoleName: aws.String("some-role-name-openshift"),
					},
				},
			}, nil)
		mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), gomock.Any()).Return(
			&iam.ListRoleTagsOutput{
				IsTruncated: false,
			}, nil)
		mockIamAPI.EXPECT().ListAttachedRolePolicies(gomock.Any(), gomock.Any()).Return(
			&iam.ListAttachedRolePoliciesOutput{
				IsTruncated: false,
				AttachedPolicies: []iamtypes.AttachedPolicy{
					{
						PolicyName: aws.String("some-policy-name"),
					},
				},
			}, nil)
		mockIamAPI.EXPECT().ListPolicyTags(gomock.Any(), gomock.Any()).Return(
			&iam.ListPolicyTagsOutput{
				IsTruncated: false,
				Tags: []iamtypes.Tag{
					{
						Key:   aws.String(common.OpenShiftVersion),
						Value: aws.String("4.13"),
					},
				},
			}, nil)
		roles, err := client.ListOperatorRoles("4.13", "", "")
		Expect(err).ToNot(HaveOccurred())
		Expect(roles).To(HaveLen(1))
	})

	It("Retrieves by target cluster ID", func() {
		mockIamAPI.EXPECT().ListRoles(gomock.Any(), gomock.Any()).Return(
			&iam.ListRolesOutput{
				IsTruncated: false,
				Roles: []iamtypes.Role{
					{
						RoleName: aws.String("some-role-name-openshift"),
					},
				},
			}, nil)
		mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), gomock.Any()).Return(
			&iam.ListRoleTagsOutput{
				IsTruncated: false,
				Tags: []iamtypes.Tag{
					{
						Key:   aws.String(tags.ClusterID),
						Value: aws.String("123"),
					},
				},
			}, nil)
		mockIamAPI.EXPECT().ListAttachedRolePolicies(gomock.Any(), gomock.Any()).Return(
			&iam.ListAttachedRolePoliciesOutput{
				IsTruncated: false,
				AttachedPolicies: []iamtypes.AttachedPolicy{
					{
						PolicyName: aws.String("some-policy-name"),
					},
				},
			}, nil)
		mockIamAPI.EXPECT().ListPolicyTags(gomock.Any(), gomock.Any()).Return(
			&iam.ListPolicyTagsOutput{
				IsTruncated: false,
				Tags: []iamtypes.Tag{
					{
						Key:   aws.String(common.OpenShiftVersion),
						Value: aws.String("4.13"),
					},
				},
			}, nil)
		roles, err := client.ListOperatorRoles("", "123", "")
		Expect(err).ToNot(HaveOccurred())
		Expect(roles).To(HaveLen(1))
	})

	It("Retrieves by target prefix", func() {
		mockIamAPI.EXPECT().ListRoles(gomock.Any(), gomock.Any()).Return(
			&iam.ListRolesOutput{
				IsTruncated: false,
				Roles: []iamtypes.Role{
					{
						RoleName: aws.String("some-role-name-openshift"),
					},
				},
			}, nil)
		mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), gomock.Any()).Return(
			&iam.ListRoleTagsOutput{
				IsTruncated: false,
				Tags: []iamtypes.Tag{
					{
						Key:   aws.String(common.ManagedPolicies),
						Value: aws.String("true"),
					}},
			}, nil)
		mockIamAPI.EXPECT().ListAttachedRolePolicies(gomock.Any(), gomock.Any()).Return(
			&iam.ListAttachedRolePoliciesOutput{
				IsTruncated: false,
				AttachedPolicies: []iamtypes.AttachedPolicy{
					{
						PolicyName: aws.String("some-policy-name"),
					},
				},
			}, nil)
		roles, err := client.ListOperatorRoles("", "", "some-role-name")
		Expect(err).ToNot(HaveOccurred())
		Expect(roles).To(HaveLen(1))
	})
})

var _ = Describe("mapToAccountRoles", func() {
	var (
		client     awsClient
		mockIamAPI *mocks.MockIamApiClient
		mockCtrl   *gomock.Controller
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockIamAPI = mocks.NewMockIamApiClient(mockCtrl)
		client = awsClient{
			iamClient: mockIamAPI,
		}
	})

	It("Skips roles that don't match version", func() {
		mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), gomock.Any()).Return(&iam.ListRoleTagsOutput{
			Tags: []iamtypes.Tag{
				{
					Key:   aws.String(common.OpenShiftVersion),
					Value: aws.String("4.13"),
				},
			},
		}, nil)
		mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), gomock.Any()).Return(&iam.ListRoleTagsOutput{
			Tags: []iamtypes.Tag{
				{
					Key:   aws.String(common.OpenShiftVersion),
					Value: aws.String("4.15"),
				},
			},
		}, nil)
		roles, err := client.mapToAccountRoles("4.13", []iamtypes.Role{
			{
				RoleName: aws.String("prefix-Installer-Role"),
			},
			{
				RoleName: aws.String("prefix2-Installer-Role"),
			},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(roles).To(HaveLen(1))
	})

	It("Retrieves all roles", func() {
		mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), gomock.Any()).Return(&iam.ListRoleTagsOutput{
			Tags: []iamtypes.Tag{
				{
					Key:   aws.String(common.OpenShiftVersion),
					Value: aws.String("4.13"),
				},
			},
		}, nil)
		mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), gomock.Any()).Return(&iam.ListRoleTagsOutput{
			Tags: []iamtypes.Tag{
				{
					Key:   aws.String(common.OpenShiftVersion),
					Value: aws.String("4.15"),
				},
			},
		}, nil)
		roles, err := client.mapToAccountRoles("", []iamtypes.Role{
			{
				RoleName: aws.String("prefix-Installer-Role"),
			},
			{
				RoleName: aws.String("prefix2-Installer-Role"),
			},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(roles).To(HaveLen(2))
	})

})

var _ = Describe("Is Policy Compatible", func() {
	var (
		client   Client
		mockCtrl *gomock.Controller

		mockEC2API            *mocks.MockEc2ApiClient
		mockCfAPI             *mocks.MockCloudFormationApiClient
		mockIamAPI            *mocks.MockIamApiClient
		mockS3API             *mocks.MockS3ApiClient
		mockSecretsManagerAPI *mocks.MockSecretsManagerApiClient
		mockSTSApi            *mocks.MockStsApiClient
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
			logrus.New(),
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
	When("Version is empty", func() {
		It("Should be compatible", func() {
			isCompatible, err := client.IsPolicyCompatible("fakearn", "")
			Expect(err).To(BeNil())
			Expect(isCompatible).To(BeTrue())
		})
	})
})

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
					Value: aws.String(TrueString),
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
					Value: aws.String(TrueString),
				},
				{
					Key:   aws.String("rosa_hcp_policies"),
					Value: aws.String(TrueString),
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
					Value: aws.String(TrueString),
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
		mockIamAPI *mocks.MockIamApiClient
		mockCtrl   *gomock.Controller
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockIamAPI = mocks.NewMockIamApiClient(mockCtrl)
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
			mockIamAPI.EXPECT().DeleteRole(gomock.Any(), &iam.DeleteRoleInput{
				RoleName: aws.String(role),
			}).Return(nil, errors.New("operator role 'test' does not exist. Skipping"))

			err := client.DeleteRole(role)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expectedErrorMessage))
		})
	})
	When("Role exists", func() {
		It("Should delete the role successfully", func() {
			role := "test"
			mockIamAPI.EXPECT().DeleteRole(gomock.Any(), &iam.DeleteRoleInput{
				RoleName: aws.String(role),
			}).Return(&iam.DeleteRoleOutput{}, nil)
			err := client.DeleteRole(role)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

var _ = Describe("Cluster Roles/Policies", func() {

	var (
		client     awsClient
		mockIamAPI *mocks.MockIamApiClient
		mockCtrl   *gomock.Controller

		accountRole           = "sample-Installer-Role"
		rolePrefix            = "acct-prefix"
		operatorRole          = "sample-operator-role"
		operatorName          = "sample-operator-name"
		operatorNameSpace     = "sample-operator-ns"
		accountRolePolicyArn  = "sample-account-role-policy-arn"
		operatorRolePolicyArn = "sample-operator-role-policy-arn"
		customPolicyArn       = "sample-custom-policy-arn"

		accountRoleAttachedPolicies = &iam.ListAttachedRolePoliciesOutput{
			AttachedPolicies: []iamtypes.AttachedPolicy{
				{
					PolicyArn: aws.String(accountRolePolicyArn),
				},
				{
					PolicyArn: aws.String(customPolicyArn),
				},
			},
		}
		operatorRoleAttachedPolicies = &iam.ListAttachedRolePoliciesOutput{
			AttachedPolicies: []iamtypes.AttachedPolicy{
				{
					PolicyArn: aws.String(operatorRolePolicyArn),
				},
				{
					PolicyArn: aws.String(customPolicyArn),
				},
			},
		}

		accountRolePolicyTags = &iam.ListPolicyTagsOutput{
			Tags: []iamtypes.Tag{
				{
					Key:   aws.String(tags.RedHatManaged),
					Value: aws.String(TrueString),
				},
				{
					Key:   aws.String(tags.RolePrefix),
					Value: aws.String(rolePrefix),
				},
			},
		}
		operatorTagArr = []iamtypes.Tag{
			{
				Key:   aws.String(tags.RedHatManaged),
				Value: aws.String(TrueString),
			},
			{
				Key:   aws.String(tags.OperatorName),
				Value: aws.String(operatorName),
			},
			{
				Key:   aws.String(tags.OperatorNamespace),
				Value: aws.String(operatorNameSpace),
			},
		}
		operatorRoleTags = &iam.ListRoleTagsOutput{
			Tags: operatorTagArr,
		}
		operatorRolePolicyTags = &iam.ListPolicyTagsOutput{
			Tags: operatorTagArr,
		}
		customPolicyTags = &iam.ListPolicyTagsOutput{
			Tags: []iamtypes.Tag{},
		}
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockIamAPI = mocks.NewMockIamApiClient(mockCtrl)
		client = awsClient{
			iamClient: mockIamAPI,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})
	It("Test getAttachedPolicies", func() {
		mockIamAPI.EXPECT().ListAttachedRolePolicies(gomock.Any(), &iam.ListAttachedRolePoliciesInput{
			RoleName: aws.String(accountRole),
		}).Return(accountRoleAttachedPolicies, nil)
		mockIamAPI.EXPECT().ListAttachedRolePolicies(gomock.Any(), &iam.ListAttachedRolePoliciesInput{
			RoleName: aws.String(operatorRole),
		}).Return(operatorRoleAttachedPolicies, nil)
		mockIamAPI.EXPECT().ListPolicyTags(gomock.Any(), &iam.ListPolicyTagsInput{
			PolicyArn: aws.String(accountRolePolicyArn),
		}).Return(accountRolePolicyTags, nil)
		mockIamAPI.EXPECT().ListPolicyTags(gomock.Any(), &iam.ListPolicyTagsInput{
			PolicyArn: aws.String(operatorRolePolicyArn),
		}).Return(operatorRolePolicyTags, nil)
		mockIamAPI.EXPECT().ListPolicyTags(gomock.Any(), &iam.ListPolicyTagsInput{
			PolicyArn: aws.String(customPolicyArn),
		}).Return(customPolicyTags, nil).MaxTimes(2)
		mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), &iam.ListRoleTagsInput{
			RoleName: aws.String(operatorName),
		}).Return(operatorRoleTags, nil)

		policies, _, err := getAttachedPolicies(mockIamAPI, accountRole, getAcctRolePolicyTags(rolePrefix))
		Expect(err).NotTo(HaveOccurred())
		Expect(policies).To(HaveLen(1))
		Expect(policies[0]).To(Equal(accountRolePolicyArn))

		tagFilter, err := getOperatorRolePolicyTags(mockIamAPI, operatorName)
		Expect(err).NotTo(HaveOccurred())
		policies, _, err = getAttachedPolicies(mockIamAPI, operatorRole, tagFilter)
		Expect(err).NotTo(HaveOccurred())
		Expect(policies).To(HaveLen(1))
		Expect(policies[0]).To(Equal(operatorRolePolicyArn))
	})
	It("Test GetPolicyDetailsFromRole", func() {
		mockIamAPI.EXPECT().ListAttachedRolePolicies(gomock.Any(), &iam.ListAttachedRolePoliciesInput{
			RoleName: aws.String(accountRole),
		}).Return(accountRoleAttachedPolicies, nil).Times(1)
		mockIamAPI.EXPECT().GetPolicy(gomock.Any(), gomock.Any()).Times(2)
		output, err := client.GetPolicyDetailsFromRole(&accountRole)
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(HaveLen(2))
	})
	It("Test DeleteAccountRole", func() {
		mockIamAPI.EXPECT().ListRolePolicies(gomock.Any(), gomock.Any()).Return(&iam.ListRolePoliciesOutput{}, nil)
		mockIamAPI.EXPECT().ListAttachedRolePolicies(gomock.Any(), &iam.ListAttachedRolePoliciesInput{
			RoleName: aws.String(accountRole),
		}).Return(accountRoleAttachedPolicies, nil).MaxTimes(2)
		mockIamAPI.EXPECT().ListPolicyTags(gomock.Any(), &iam.ListPolicyTagsInput{
			PolicyArn: aws.String(accountRolePolicyArn),
		}).Return(accountRolePolicyTags, nil)
		mockIamAPI.EXPECT().ListPolicyTags(gomock.Any(), &iam.ListPolicyTagsInput{
			PolicyArn: aws.String(customPolicyArn),
		}).Return(customPolicyTags, nil)
		mockIamAPI.EXPECT().DetachRolePolicy(gomock.Any(), &iam.DetachRolePolicyInput{
			RoleName:  aws.String(accountRole),
			PolicyArn: aws.String(accountRolePolicyArn),
		}).Return(&iam.DetachRolePolicyOutput{}, nil)
		mockIamAPI.EXPECT().DetachRolePolicy(gomock.Any(), &iam.DetachRolePolicyInput{
			RoleName:  aws.String(accountRole),
			PolicyArn: aws.String(customPolicyArn),
		}).Return(&iam.DetachRolePolicyOutput{}, nil)
		mockIamAPI.EXPECT().DeleteRole(gomock.Any(), &iam.DeleteRoleInput{
			RoleName: aws.String(accountRole),
		}).Return(&iam.DeleteRoleOutput{}, nil)
		attachCount := int32(0)
		mockIamAPI.EXPECT().GetPolicy(gomock.Any(), &iam.GetPolicyInput{
			PolicyArn: aws.String(accountRolePolicyArn),
		}).Return(&iam.GetPolicyOutput{
			Policy: &iamtypes.Policy{
				AttachmentCount: &attachCount,
			},
		}, nil)
		mockIamAPI.EXPECT().ListPolicyVersions(gomock.Any(), gomock.Any()).Return(&iam.ListPolicyVersionsOutput{
			Versions: []iamtypes.PolicyVersion{},
		}, nil)
		mockIamAPI.EXPECT().DeletePolicy(gomock.Any(), &iam.DeletePolicyInput{
			PolicyArn: aws.String(accountRolePolicyArn),
		}).Return(&iam.DeletePolicyOutput{}, nil)
		err := client.DeleteAccountRole(accountRole, rolePrefix, false, false)
		Expect(err).NotTo(HaveOccurred())
	})
	It("Test DeleteOperatorRole", func() {
		mockIamAPI.EXPECT().ListAttachedRolePolicies(gomock.Any(), &iam.ListAttachedRolePoliciesInput{
			RoleName: aws.String(operatorRole),
		}).Return(operatorRoleAttachedPolicies, nil).MaxTimes(2)
		mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), &iam.ListRoleTagsInput{
			RoleName: aws.String(operatorRole),
		}).Return(operatorRoleTags, nil)
		mockIamAPI.EXPECT().ListPolicyTags(gomock.Any(), &iam.ListPolicyTagsInput{
			PolicyArn: aws.String(operatorRolePolicyArn),
		}).Return(operatorRolePolicyTags, nil)
		mockIamAPI.EXPECT().ListPolicyTags(gomock.Any(), &iam.ListPolicyTagsInput{
			PolicyArn: aws.String(customPolicyArn),
		}).Return(customPolicyTags, nil)
		mockIamAPI.EXPECT().DetachRolePolicy(gomock.Any(), &iam.DetachRolePolicyInput{
			RoleName:  aws.String(operatorRole),
			PolicyArn: aws.String(operatorRolePolicyArn),
		}).Return(&iam.DetachRolePolicyOutput{}, nil)
		mockIamAPI.EXPECT().DetachRolePolicy(gomock.Any(), &iam.DetachRolePolicyInput{
			RoleName:  aws.String(operatorRole),
			PolicyArn: aws.String(customPolicyArn),
		}).Return(&iam.DetachRolePolicyOutput{}, nil)
		mockIamAPI.EXPECT().DeleteRole(gomock.Any(), &iam.DeleteRoleInput{
			RoleName: aws.String(operatorRole),
		}).Return(&iam.DeleteRoleOutput{}, nil)
		attachCount := int32(1)
		mockIamAPI.EXPECT().GetPolicy(gomock.Any(), &iam.GetPolicyInput{
			PolicyArn: aws.String(operatorRolePolicyArn),
		}).Return(&iam.GetPolicyOutput{
			Policy: &iamtypes.Policy{
				AttachmentCount: &attachCount,
			},
		}, nil)
		_, err := client.DeleteOperatorRole(operatorRole, false, false)
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("Validates isAwsManagedPolicy function", func() {
	var (
		awsManagedPolicyArn = "arn:aws:iam::aws:policy/service-role/ROSAInstallerPolicy"
		customPolicyArn     = "arn:aws:iam::765374464689:policy/test-policy"
	)
	It("check aws managed policy", func() {
		result := isAwsManagedPolicy(awsManagedPolicyArn)
		Expect(result).To(Equal(true))
	})
	It("check custom policy", func() {
		result := isAwsManagedPolicy(customPolicyArn)
		Expect(result).To(Equal(false))
	})
})

var _ = Describe("CheckIfROSAOperatorRole", func() {

	var (
		credRequest map[string]*cmv1.STSOperator
		role        iamtypes.Role
		result      bool
	)

	BeforeEach(func() {
		stsOperator1, err := cmv1.NewSTSOperator().Namespace("namespace-1").Build()
		Expect(err).NotTo(HaveOccurred())
		stsOperator2, err := cmv1.NewSTSOperator().Namespace("namespace-2").Build()
		Expect(err).NotTo(HaveOccurred())
		credRequest = map[string]*cmv1.STSOperator{
			"operator-1": stsOperator1,
			"operator-2": stsOperator2,
		}
	})

	When("the role has matching tags", func() {
		It("should return true", func() {
			role = iamtypes.Role{
				RoleName: aws.String("test-role-name"),
				Tags: []iamtypes.Tag{
					{
						Key:   aws.String("operator_namespace"),
						Value: aws.String("namespace-1"),
					},
				},
			}
			result = checkIfROSAOperatorRole(role, credRequest)
			Expect(result).To(BeTrue())
		})
	})

	Context("the role name contains the namespace", func() {
		BeforeEach(func() {
			role = iamtypes.Role{
				RoleName: aws.String("test-role-namespace-2"),
				Tags:     []iamtypes.Tag{},
			}
			result = checkIfROSAOperatorRole(role, credRequest)
		})

		It("should return true", func() {
			role = iamtypes.Role{
				RoleName: aws.String("test-role-namespace-2"),
				Tags:     []iamtypes.Tag{},
			}
			result = checkIfROSAOperatorRole(role, credRequest)
			Expect(result).To(BeTrue())
		})
	})

	When("the role has no matching tags or name", func() {
		It("should return false", func() {
			role = iamtypes.Role{
				RoleName: aws.String("test-role-name"),
				Tags: []iamtypes.Tag{
					{
						Key:   aws.String("operator_namespace"),
						Value: aws.String("non-matching-namespace"),
					},
				},
			}
			result = checkIfROSAOperatorRole(role, credRequest)
			Expect(result).To(BeFalse())
		})
	})
})

var _ = Describe("doesPolicyHaveTags", func() {
	var (
		mockIamAPI *mocks.MockIamApiClient
		mockCtrl   *gomock.Controller
	)
	testRoleArn := "fake-role-arn"
	testePolicyArn := "fake-policy-arn"
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockIamAPI = mocks.NewMockIamApiClient(mockCtrl)
	})
	It("Should have a red-hat-managed policy identified", func() {
		mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), gomock.Any()).Return(&iam.ListRoleTagsOutput{
			Tags: []iamtypes.Tag{
				{
					Key:   aws.String(tags.OperatorName),
					Value: aws.String("ebs-cloud-credentials"),
				},
				{
					Key:   aws.String(tags.OperatorNamespace),
					Value: aws.String("openshift-cluster-csi-drivers"),
				},
				{
					Key:   aws.String(tags.RedHatManaged),
					Value: aws.String("true"),
				},
			},
			IsTruncated: false,
		}, nil)
		filter, err := getOperatorRolePolicyTags(mockIamAPI, testRoleArn)
		Expect(err).ToNot(HaveOccurred())
		Expect(filter).To(HaveLen(3))
		mockIamAPI.EXPECT().ListPolicyTags(gomock.Any(), gomock.Any()).Return(&iam.ListPolicyTagsOutput{
			Tags: []iamtypes.Tag{
				{
					Key:   aws.String(tags.OperatorName),
					Value: aws.String("ebs-cloud-credentials"),
				},
				{
					Key:   aws.String(tags.OperatorNamespace),
					Value: aws.String("openshift-cluster-csi-drivers"),
				},
				{
					Key:   aws.String(tags.RedHatManaged),
					Value: aws.String("true"),
				},
				{
					Key:   aws.String("t1"),
					Value: aws.String("v1"),
				},
				{
					Key:   aws.String("t2"),
					Value: aws.String("v2"),
				},
				{
					Key:   aws.String("t3"),
					Value: aws.String("v3"),
				},
				{
					Key:   aws.String("t4"),
					Value: aws.String("v4"),
				},
			},
			IsTruncated: false,
		}, nil)
		result, err := doesPolicyHaveTags(mockIamAPI, &testePolicyArn, filter)
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeTrue())
	})
	It("Should not have a red-hat-managed policy identified", func() {
		mockIamAPI.EXPECT().ListRoleTags(gomock.Any(), gomock.Any()).Return(&iam.ListRoleTagsOutput{
			Tags: []iamtypes.Tag{
				{
					Key:   aws.String(tags.OperatorName),
					Value: aws.String("ebs-cloud-credentials"),
				},
				{
					Key:   aws.String(tags.OperatorNamespace),
					Value: aws.String("openshift-cluster-csi-drivers"),
				},
				{
					Key:   aws.String(tags.RedHatManaged),
					Value: aws.String("true"),
				},
			},
			IsTruncated: false,
		}, nil)
		filter, err := getOperatorRolePolicyTags(mockIamAPI, testRoleArn)
		Expect(err).ToNot(HaveOccurred())
		Expect(filter).To(HaveLen(3))
		mockIamAPI.EXPECT().ListPolicyTags(gomock.Any(), gomock.Any()).Return(&iam.ListPolicyTagsOutput{
			Tags: []iamtypes.Tag{
				{
					Key:   aws.String("t1"),
					Value: aws.String("v1"),
				},
				{
					Key:   aws.String("t2"),
					Value: aws.String("v2"),
				},
				{
					Key:   aws.String("t3"),
					Value: aws.String("v3"),
				},
				{
					Key:   aws.String("t4"),
					Value: aws.String("v4"),
				},
			},
			IsTruncated: false,
		}, nil)
		result, err := doesPolicyHaveTags(mockIamAPI, &testePolicyArn, filter)
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeFalse())
	})
	It("Considers the policy have the tags as the filters are empty", func() {
		result, err := doesPolicyHaveTags(mockIamAPI, &testePolicyArn, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeTrue())
	})
})

var _ = Describe("validateManagedPolicy", func() {
	var (
		client                awsClient
		mockIamAPI            *mocks.MockIamApiClient
		mockCtrl              *gomock.Controller
		ec2ContainerPolicy, _ = (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::ec2Container").Build()
		workerPolicy, _       = (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::worker").Build()
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockIamAPI = mocks.NewMockIamApiClient(mockCtrl)
		client = awsClient{
			iamClient: mockIamAPI,
			logger:    logging.NewLogger(),
		}
	})

	DescribeTable("validate ECR policy", func(
		policies map[string]*cmv1.AWSSTSPolicy, policyKey, roleName, expectedErr string,
	) {
		mockIamAPI.EXPECT().ListAttachedRolePolicies(gomock.Any(), gomock.Any()).Return(nil, nil).Times(0)
		err := client.validateManagedPolicy(policies, policyKey, roleName)
		if expectedErr == "" {
			Expect(err).To(BeNil())
		} else {
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring(expectedErr))
		}
	},
		Entry("succeeds if ECR policy does not exist", map[string]*cmv1.AWSSTSPolicy{
			"sts_hcp_instance_worker_permission_policy": workerPolicy},
			"sts_hcp_ec2_registry_permission_policy", "worker", ""),
		Entry("succeeds if ECR policy exist but skips check if policy is attached",
			map[string]*cmv1.AWSSTSPolicy{
				"sts_hcp_instance_worker_permission_policy": workerPolicy,
				"sts_hcp_ec2_registry_permission_policy":    ec2ContainerPolicy},
			"sts_hcp_ec2_registry_permission_policy", "worker", ""),
		Entry("fails to find worker policy", map[string]*cmv1.AWSSTSPolicy{
			"sts_hcp_ec2_registry_permission_policy": ec2ContainerPolicy},
			"sts_hcp_instance_worker_permission_policy", "worker",
			"failed to find policy ARN for 'sts_hcp_instance_worker_permission_policy'"),
	)
})

var _ = Describe("ListPolicyVersions", func() {
	var (
		mockIamAPI *mocks.MockIamApiClient
		mockCtrl   *gomock.Controller
		client     *awsClient
		policyArn  string
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockIamAPI = mocks.NewMockIamApiClient(mockCtrl)
		client = &awsClient{iamClient: mockIamAPI}
		policyArn = "arn:aws:iam::123456789012:policy/test-policy"
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("Should list policy versions successfully", func() {
		expectedVersions := []iamtypes.PolicyVersion{
			{
				VersionId:        aws.String("v1"),
				IsDefaultVersion: true,
			},
			{
				VersionId:        aws.String("v2"),
				IsDefaultVersion: false,
			},
		}

		mockIamAPI.EXPECT().ListPolicyVersions(gomock.Any(), &iam.ListPolicyVersionsInput{
			PolicyArn: aws.String(policyArn),
		}).Return(&iam.ListPolicyVersionsOutput{
			Versions: expectedVersions,
		}, nil)

		result, err := client.ListPolicyVersions(policyArn)
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(HaveLen(2))
		Expect(result[0].VersionID).To(Equal("v1"))
		Expect(result[0].IsDefaultVersion).To(BeTrue())
		Expect(result[1].VersionID).To(Equal("v2"))
		Expect(result[1].IsDefaultVersion).To(BeFalse())
	})

	It("Should return an error when ListPolicyVersions fails", func() {
		mockIamAPI.EXPECT().ListPolicyVersions(gomock.Any(), &iam.ListPolicyVersionsInput{
			PolicyArn: aws.String(policyArn),
		}).Return(nil, fmt.Errorf("failed to list policy versions"))

		result, err := client.ListPolicyVersions(policyArn)
		Expect(err).To(HaveOccurred())
		Expect(result).To(BeNil())
	})
})
