package accountroles

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"go.uber.org/mock/gomock"

	"github.com/aws/aws-sdk-go-v2/aws"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/ststrust"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	mock "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	testExternalID  = "223B9588-36A5-ECA4-BE8D-7C673B77CEC1"
	otherExternalID = "A1B2C3D4-E5F6-7890-ABCD-EF1234567890"
)

const (
	hcpInstallerRoleName = "test-HCP-ROSA-Installer-Role"
	hcpSupportRoleName   = "test-HCP-ROSA-Support-Role"
)

var _ = Describe("Accountroles", Ordered, func() {
	var (
		mockCtrl   *gomock.Controller
		mockClient *mock.MockClient
		r          *rosa.Runtime
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mock.NewMockClient(mockCtrl)
		r = rosa.NewRuntime()
		r.AWSClient = mockClient
		r.Creator = &mock.Creator{ARN: "arn-123"}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	When("createRoles", func() {
		It("createRole fails to find policy ARN", func() {
			mockClient.EXPECT().CheckRoleExists(gomock.Any()).Return(false, "", nil).AnyTimes()
			mockClient.EXPECT().EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return("role-123", nil).AnyTimes()

			policies := map[string]*cmv1.AWSSTSPolicy{}
			accountRolesCreationInput := buildRolesCreationInput("test", "", "account-123", "stage", policies, "", "", false, "")
			err := (&hcpManagedPoliciesCreator{}).createRoles(r, accountRolesCreationInput)
			Expect(err).To(HaveOccurred(), "createRoles should fail when managed policy ARNs are missing")
			Expect(err.Error()).To(ContainSubstring("failed to find policy ARN for"),
				"error should identify the missing managed policy ARN")
		})
		It("createRole succeeds without ec2 policy", func() {
			mockClient.EXPECT().AttachRolePolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(3)
			mockClient.EXPECT().CheckRoleExists(gomock.Any()).Return(false, "", nil).AnyTimes()
			mockClient.EXPECT().EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return("role-123", nil).AnyTimes()

			installerPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::installer").Build()
			workerPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::worker").Build()
			supportPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::support").Build()

			policies := map[string]*cmv1.AWSSTSPolicy{
				"sts_hcp_installer_permission_policy":       installerPolicy,
				"sts_hcp_instance_worker_permission_policy": workerPolicy,
				"sts_hcp_support_permission_policy":         supportPolicy,
			}

			accountRolesCreationInput := buildRolesCreationInput("test", "", "account-123", "stage", policies, "", "", false, "")
			err := (&hcpManagedPoliciesCreator{}).createRoles(r, accountRolesCreationInput)
			Expect(err).ToNot(HaveOccurred(), "createRoles should succeed without optional ec2 policy")
		})
		It("createRole succeeds when ec2 policy is available but not attached", func() {
			mockClient.EXPECT().AttachRolePolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(3)
			mockClient.EXPECT().CheckRoleExists(gomock.Any()).Return(false, "", nil).AnyTimes()
			mockClient.EXPECT().EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return("role-123", nil).AnyTimes()

			installerPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::installer").Build()
			workerPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::worker").Build()
			supportPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::support").Build()
			ec2Policy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::ec2").Build()

			policies := map[string]*cmv1.AWSSTSPolicy{
				"sts_hcp_installer_permission_policy":       installerPolicy,
				"sts_hcp_instance_worker_permission_policy": workerPolicy,
				"sts_hcp_support_permission_policy":         supportPolicy,
				"sts_hcp_ec2_registry_permission_policy":    ec2Policy,
			}

			accountRolesCreationInput := buildRolesCreationInput("test", "", "account-123", "stage", policies, "", "", false, "")
			err := (&hcpManagedPoliciesCreator{}).createRoles(r, accountRolesCreationInput)
			Expect(err).ToNot(HaveOccurred(), "createRoles should succeed when ec2 policy is present but unused")
		})
		It("createRole succeeds with hosted-cp and shared-vpc roles", func() {
			mockClient.EXPECT().AttachRolePolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(5)
			mockClient.EXPECT().CheckRoleExists(gomock.Any()).Return(false, "", nil).AnyTimes()
			mockClient.EXPECT().EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return("arn::role:role-123", nil).AnyTimes()
			mockClient.EXPECT().EnsurePolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any()).Return("arn::policy:123", nil).Times(2)

			installerPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::installer").Build()
			workerPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::worker").Build()
			supportPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::support").Build()
			ec2Policy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::ec2").Build()

			policies := map[string]*cmv1.AWSSTSPolicy{
				"sts_hcp_installer_permission_policy":       installerPolicy,
				"sts_hcp_instance_worker_permission_policy": workerPolicy,
				"sts_hcp_support_permission_policy":         supportPolicy,
				"sts_hcp_ec2_registry_permission_policy":    ec2Policy,
			}

			accountRolesCreationInput := buildRolesCreationInput("test", "123", "account-123", "stage", policies,
				"123", "123", true, "")
			args.route53RoleArn = "arn:aws:iam::123:role/route53"
			args.vpcEndpointRoleArn = "arn:aws:iam::123:role/vpce"
			err := (&hcpManagedPoliciesCreator{}).createRoles(r, accountRolesCreationInput)
			Expect(err).ToNot(HaveOccurred(), "createRoles should succeed for hosted-cp shared VPC configuration")
		})
		It("createRole succeeds with hosted-cp and govcloud env", func() {
			mockClient.EXPECT().AttachRolePolicy(gomock.Any(), "test-HCP-ROSA-Installer-Role", "arn::installer").Times(1)
			mockClient.EXPECT().AttachRolePolicy(gomock.Any(), "test-HCP-ROSA-Support-Role", "arn::support").Times(1)
			mockClient.EXPECT().AttachRolePolicy(gomock.Any(), "test-HCP-ROSA-Worker-Role", "arn::worker").Times(1)
			mockClient.EXPECT().CheckRoleExists(gomock.Any()).Return(false, "", nil).AnyTimes()
			mockClient.EXPECT().EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return("arn::role:role-123", nil).AnyTimes()

			r.Creator.IsGovcloud = true
			installerPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::installer").Build()
			workerPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::worker").Build()
			supportPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::support").Build()

			policies := map[string]*cmv1.AWSSTSPolicy{
				"sts_hcp_installer_permission_policy":       installerPolicy,
				"sts_hcp_instance_worker_permission_policy": workerPolicy,
				"sts_hcp_support_permission_policy":         supportPolicy,
			}

			accountRolesCreationInput := buildRolesCreationInput("test", "mock-permissions-boundary", "account-123", "stage", policies,
				"123", "mock-path", false, "")
			err := (&hcpManagedPoliciesCreator{}).createRoles(r, accountRolesCreationInput)
			Expect(err).ToNot(HaveOccurred(), "createRoles should succeed in govcloud hosted-cp configuration")
		})
	})

	When("existing account roles with external-id", func() {
		hcpPolicies := func() map[string]*cmv1.AWSSTSPolicy {
			installerPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::installer").Build()
			workerPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::worker").Build()
			supportPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::support").Build()
			return map[string]*cmv1.AWSSTSPolicy{
				"sts_hcp_installer_permission_policy":       installerPolicy,
				"sts_hcp_instance_worker_permission_policy": workerPolicy,
				"sts_hcp_support_permission_policy":         supportPolicy,
			}
		}

		checkRoleExistsForHCP := func(installerExists, supportExists bool) func(string) (bool, string, error) {
			return func(name string) (bool, string, error) {
				switch name {
				case hcpInstallerRoleName:
					if installerExists {
						return true, "arn:aws:iam::123:role/installer", nil
					}
				case hcpSupportRoleName:
					if supportExists {
						return true, "arn:aws:iam::123:role/support", nil
					}
				}
				return false, "", nil
			}
		}

		It("createRole succeeds when existing installer and support trust policies match external-id", func() {
			mockClient.EXPECT().AttachRolePolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(3)
			mockClient.EXPECT().CheckRoleExists(gomock.Any()).DoAndReturn(
				checkRoleExistsForHCP(true, true)).AnyTimes()
			mockClient.EXPECT().GetRoleByName(gomock.Any()).DoAndReturn(func(name string) (iamtypes.Role, error) {
				return roleWithTrustPolicy(policyWithExternalID(testExternalID)), nil
			}).AnyTimes()
			mockClient.EXPECT().EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return("role-123", nil).Times(3)

			accountRolesCreationInput := buildRolesCreationInput(
				"test", "", "account-123", "stage", hcpPolicies(), "", "", false, testExternalID,
			)
			err := (&hcpManagedPoliciesCreator{}).createRoles(r, accountRolesCreationInput)
			Expect(err).ToNot(HaveOccurred(),
				"createRoles should succeed when existing installer and support trust policies match --external-id")
		})

		It("getAssumeRolePolicy returns the existing trust policy when the template is empty", func() {
			existingPolicy := policyWithExternalID(testExternalID)
			mockClient.EXPECT().CheckRoleExists(hcpInstallerRoleName).Return(true, "arn:aws:iam::123:role/installer", nil)
			mockClient.EXPECT().GetRoleByName(hcpInstallerRoleName).Return(
				roleWithTrustPolicy(existingPolicy), nil,
			)

			accountRolesCreationInput := buildRolesCreationInput(
				"test", "", "account-123", "stage", hcpPolicies(), "", "", false, testExternalID,
			)
			policy, err := getAssumeRolePolicy(r, "aws", mock.HCPInstallerRole, hcpInstallerRoleName, accountRolesCreationInput)
			Expect(err).ToNot(HaveOccurred(),
				"getAssumeRolePolicy should fall back to the existing trust policy when the template is empty")
			ids, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(policy)
			Expect(err).ToNot(HaveOccurred(), "failed to parse fallback trust policy")
			Expect(ids).To(Equal([]string{testExternalID}),
				"fallback trust policy should preserve the existing external-id")
		})

		It("getAssumeRolePolicy preserves multiple external-ids when none were requested", func() {
			otherID := "333B9588-36A5-ECA4-BE8D-7C673B77CDCD"
			existingPolicy := policyWithMultipleExternalIDs([]string{testExternalID, otherID})
			mockClient.EXPECT().CheckRoleExists(hcpInstallerRoleName).Return(true, "arn:aws:iam::123:role/installer", nil)
			mockClient.EXPECT().GetRoleByName(hcpInstallerRoleName).Return(
				roleWithTrustPolicy(existingPolicy), nil,
			)

			trustTemplate, _ := (&cmv1.AWSSTSPolicyBuilder{}).Details(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Principal": { "AWS": "arn:aws:iam::123:role/test" },
					"Action": "sts:AssumeRole"
				}]
			}`).Build()
			policies := hcpPolicies()
			policies["sts_installer_trust_policy"] = trustTemplate

			accountRolesCreationInput := buildRolesCreationInput(
				"test", "", "account-123", "stage", policies, "", "", false, "",
			)
			policy, err := getAssumeRolePolicy(r, "aws", mock.HCPInstallerRole, hcpInstallerRoleName, accountRolesCreationInput)
			Expect(err).ToNot(HaveOccurred(),
				"getAssumeRolePolicy should preserve multiple existing external-ids without --external-id")
			ids, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(policy)
			Expect(err).ToNot(HaveOccurred(), "failed to parse preserved installer trust policy")
			Expect(ids).To(ConsistOf(testExternalID, otherID),
				"preserved installer trust policy should keep all existing external-ids")
		})

		It("getAssumeRolePolicy preserves external-id when rebuilding an existing installer role", func() {
			mockClient.EXPECT().CheckRoleExists(hcpInstallerRoleName).Return(true, "arn:aws:iam::123:role/installer", nil)
			mockClient.EXPECT().GetRoleByName(hcpInstallerRoleName).Return(
				roleWithTrustPolicy(policyWithExternalID(testExternalID)), nil,
			)

			accountRolesCreationInput := buildRolesCreationInput(
				"test", "", "account-123", "stage", hcpPolicies(), "", "", false, testExternalID,
			)
			policy, err := getAssumeRolePolicy(r, "aws", mock.HCPInstallerRole, hcpInstallerRoleName, accountRolesCreationInput)
			Expect(err).ToNot(HaveOccurred(),
				"getAssumeRolePolicy should rebuild an existing installer role with a matching external-id")
			ids, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(policy)
			Expect(err).ToNot(HaveOccurred(), "failed to parse rebuilt installer trust policy")
			Expect(ids).To(Equal([]string{testExternalID}),
				"rebuilt installer trust policy should include the validated external-id")
		})

		It("getAssumeRolePolicy fails when existing installer role trust policy has no external-id", func() {
			mockClient.EXPECT().CheckRoleExists(hcpInstallerRoleName).Return(true, "arn:aws:iam::123:role/installer", nil)
			mockClient.EXPECT().GetRoleByName(hcpInstallerRoleName).Return(
				roleWithTrustPolicy(policyWithoutExternalID()), nil,
			)

			accountRolesCreationInput := buildRolesCreationInput(
				"test", "", "account-123", "stage", hcpPolicies(), "", "", false, testExternalID,
			)
			_, err := getAssumeRolePolicy(r, "aws", mock.HCPInstallerRole, hcpInstallerRoleName, accountRolesCreationInput)
			Expect(err).To(HaveOccurred(),
				"getAssumeRolePolicy should fail when an existing installer role lacks sts:ExternalId")
			Expect(err.Error()).To(ContainSubstring("has no sts:ExternalId"),
				"error should explain the existing role has no sts:ExternalId condition")
		})

		It("getAssumeRolePolicy fails when existing support role trust policy has a different external-id", func() {
			mockClient.EXPECT().CheckRoleExists(hcpSupportRoleName).Return(true, "arn:aws:iam::123:role/support", nil)
			mockClient.EXPECT().GetRoleByName(hcpSupportRoleName).Return(
				roleWithTrustPolicy(policyWithExternalID(otherExternalID)), nil,
			)

			accountRolesCreationInput := buildRolesCreationInput(
				"test", "", "account-123", "stage", hcpPolicies(), "", "", false, testExternalID,
			)
			_, err := getAssumeRolePolicy(r, "aws", mock.HCPSupportRole, hcpSupportRoleName, accountRolesCreationInput)
			Expect(err).To(HaveOccurred(),
				"getAssumeRolePolicy should fail when an existing support role trust policy has a different external-id")
			Expect(errors.Is(err, ststrust.ErrExternalIDConflictOnInject)).To(BeTrue(),
				"error should report incompatible existing trust policy external-id")
		})
	})
})

// roleWithTrustPolicy builds an IAM role stub with a URL-encoded assume-role policy document.
func roleWithTrustPolicy(policyJSON string) iamtypes.Role {
	encoded := url.QueryEscape(policyJSON)
	return iamtypes.Role{
		RoleName:                 aws.String("test-role"),
		AssumeRolePolicyDocument: aws.String(encoded),
	}
}

// policyWithExternalID returns a trust policy JSON document containing the given sts:ExternalId.
func policyWithExternalID(externalID string) string {
	return `{
		"Version": "2012-10-17",
		"Statement": [{
			"Effect": "Allow",
			"Principal": { "AWS": "arn:aws:iam::123:role/test" },
			"Action": "sts:AssumeRole",
			"Condition": {
				"StringEquals": { "sts:ExternalId": "` + externalID + `" }
			}
		}]
	}`
}

// policyWithoutExternalID returns a trust policy JSON document with no sts:ExternalId condition.
func policyWithMultipleExternalIDs(externalIDs []string) string {
	encoded, err := json.Marshal(externalIDs)
	Expect(err).NotTo(HaveOccurred(), "failed to marshal external-id list for test trust policy")
	return strings.Replace(`{
		"Version": "2012-10-17",
		"Statement": [{
			"Effect": "Allow",
			"Principal": { "AWS": "arn:aws:iam::123:role/test" },
			"Action": "sts:AssumeRole",
			"Condition": {
				"StringEquals": { "sts:ExternalId": EXTERNAL_IDS }
			}
		}]
	}`, "EXTERNAL_IDS", string(encoded), 1)
}

func policyWithoutExternalID() string {
	return `{
		"Version": "2012-10-17",
		"Statement": [{
			"Effect": "Allow",
			"Principal": { "AWS": "arn:aws:iam::123:role/test" },
			"Action": "sts:AssumeRole"
		}]
	}`
}
