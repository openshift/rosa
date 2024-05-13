package aws

import (
	"errors"
	"fmt"

	gomock "go.uber.org/mock/gomock"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/aws/mocks"
	"github.com/openshift/rosa/pkg/aws/tags"
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
		err := client.DeleteAccountRole(accountRole, rolePrefix, false)
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
		err := client.DeleteOperatorRole(operatorRole, false)
		Expect(err).NotTo(HaveOccurred())
	})
})
