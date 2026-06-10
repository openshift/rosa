package aws

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strings"

	"go.uber.org/mock/gomock"

	"github.com/aws/aws-sdk-go-v2/aws"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/ststrust"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/reporter"
)

const testExternalID = "223B9588-36A5-ECA4-BE8D-7C673B77CEC1"

var installerTrustTemplate = `{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": { "AWS": "arn:aws:iam::%{aws_account_id}:role/RH-Managed-OpenShift-Installer" },
    "Action": "sts:AssumeRole"
  }]
}`

var _ = Describe("STS external ID helpers", func() {
	Describe("BuildAccountRoleAssumeRolePolicy", func() {
		It("injects ExternalId for new installer roles when external ID is set", func() {
			policies := map[string]*cmv1.AWSSTSPolicy{
				"sts_installer_trust_policy": mustPolicy(installerTrustTemplate),
			}
			policy, err := BuildAccountRoleAssumeRolePolicy(
				InstallerAccountRole, "aws", policies, "stage", testExternalID,
			)
			Expect(err).NotTo(HaveOccurred(), "failed to build installer trust policy for new role")
			ids, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(policy)
			Expect(err).NotTo(HaveOccurred(), "failed to parse injected installer trust policy")
			Expect(ids).To(Equal([]string{testExternalID}),
				"new installer trust policy should contain the provided external-id")
		})

		It("injects ExternalId when rebuilding installer trust policy with a validated external ID", func() {
			policies := map[string]*cmv1.AWSSTSPolicy{
				"sts_installer_trust_policy": mustPolicy(installerTrustTemplate),
			}
			policy, err := BuildAccountRoleAssumeRolePolicy(
				InstallerAccountRole, "aws", policies, "stage", testExternalID,
			)
			Expect(err).NotTo(HaveOccurred(), "failed to build installer trust policy for existing role")
			ids, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(policy)
			Expect(err).NotTo(HaveOccurred(), "failed to parse rebuilt installer trust policy")
			Expect(ids).To(Equal([]string{testExternalID}),
				"rebuilt installer trust policy should retain the validated external-id")
		})

		It("preserves multiple ExternalIds from an existing trust policy", func() {
			policies := map[string]*cmv1.AWSSTSPolicy{
				"sts_installer_trust_policy": mustPolicy(installerTrustTemplate),
			}
			otherID := "333B9588-36A5-ECA4-BE8D-7C673B77CDCD"
			built, err := BuildAccountRoleAssumeRolePolicy(
				InstallerAccountRole, "aws", policies, "stage", "",
			)
			Expect(err).NotTo(HaveOccurred(), "failed to build installer trust policy template")
			existing := policyWithMultipleExternalIDs([]string{testExternalID, otherID})
			preserved, err := PreserveSTSExternalIDsInTrustPolicy(built, existing)
			Expect(err).NotTo(HaveOccurred(), "failed to preserve multiple external-ids in rebuilt trust policy")
			ids, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(preserved)
			Expect(err).NotTo(HaveOccurred(), "failed to parse preserved installer trust policy")
			Expect(ids).To(ConsistOf(testExternalID, otherID),
				"rebuilt installer trust policy should preserve all existing external-ids")
		})

		It("does not inject ExternalId for worker roles", func() {
			policies := map[string]*cmv1.AWSSTSPolicy{
				"sts_instance_worker_trust_policy": mustPolicy(installerTrustTemplate),
			}
			policy, err := BuildAccountRoleAssumeRolePolicy(
				WorkerAccountRole, "aws", policies, "stage", testExternalID,
			)
			Expect(err).NotTo(HaveOccurred(), "failed to build worker trust policy")
			ids, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(policy)
			Expect(err).NotTo(HaveOccurred(), "failed to parse worker trust policy")
			Expect(ids).To(BeEmpty(), "worker trust policy should not include sts:ExternalId")
		})
	})

	Describe("ResolveAccountRoleTrustPolicyExternalID", func() {
		var (
			mockCtrl *gomock.Controller
			client   *MockClient
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			client = NewMockClient(mockCtrl)
		})

		AfterEach(func() {
			mockCtrl.Finish()
		})

		It("returns the requested external-id after validating an existing installer role", func() {
			client.EXPECT().GetRoleByName("installer-role").Return(
				roleWithPolicy(policyWithExternalID(testExternalID)), nil,
			)
			effectiveExternalID, _, err := ResolveAccountRoleTrustPolicyExternalID(
				client, "installer-role", InstallerAccountRole, testExternalID,
			)
			Expect(err).NotTo(HaveOccurred(), "existing installer role should accept matching external-id")
			Expect(effectiveExternalID).To(Equal(testExternalID),
				"resolved external-id should equal the requested value")
		})

		It("returns the requested external-id for worker roles without reading trust policies", func() {
			effectiveExternalID, existingPolicy, err := ResolveAccountRoleTrustPolicyExternalID(
				client, "worker-role", WorkerAccountRole, testExternalID,
			)
			Expect(err).NotTo(HaveOccurred(), "worker roles should not fetch trust policies")
			Expect(effectiveExternalID).To(Equal(testExternalID), "worker role should keep requested external-id")
			Expect(existingPolicy).To(BeEmpty(), "worker role should not return an existing trust policy")
		})

		It("returns empty effective external-id when multiple external-ids exist without a request", func() {
			otherID := "333B9588-36A5-ECA4-BE8D-7C673B77CDCD"
			client.EXPECT().GetRoleByName("installer-role").Return(
				roleWithPolicy(policyWithMultipleExternalIDs([]string{testExternalID, otherID})), nil,
			)
			effectiveExternalID, _, err := ResolveAccountRoleTrustPolicyExternalID(
				client, "installer-role", InstallerAccountRole, "",
			)
			Expect(err).NotTo(HaveOccurred(), "multiple existing external-ids should not error")
			Expect(effectiveExternalID).To(BeEmpty(), "multiple existing external-ids should not auto-select one")
		})

		It("discovers a single external-id when none was requested", func() {
			client.EXPECT().GetRoleByName("installer-role").Return(
				roleWithPolicy(policyWithExternalID(testExternalID)), nil,
			)
			effectiveExternalID, _, err := ResolveAccountRoleTrustPolicyExternalID(
				client, "installer-role", InstallerAccountRole, "",
			)
			Expect(err).NotTo(HaveOccurred(), "existing installer role should expose a single external-id")
			Expect(effectiveExternalID).To(Equal(testExternalID),
				"discovered external-id should match the existing trust policy")
		})
	})

	Describe("ResolveSTSExternalIDForCluster", func() {
		var (
			mockCtrl *gomock.Controller
			client   *MockClient
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			client = NewMockClient(mockCtrl)
		})

		AfterEach(func() {
			mockCtrl.Finish()
		})

		It("returns entered ID when it matches both role trust policies", func() {
			installerPolicy := policyWithExternalID(testExternalID)
			supportPolicy := policyWithExternalID(testExternalID)
			client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/installer").Return(roleWithPolicy(installerPolicy), nil)
			client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/support").Return(roleWithPolicy(supportPolicy), nil)

			id, err := ResolveSTSExternalIDForCluster(
				testExternalID,
				"arn:aws:iam::123:role/installer",
				"arn:aws:iam::123:role/support",
				client,
			)
			Expect(err).NotTo(HaveOccurred(), "cluster external-id resolution should succeed when both roles match")
			Expect(id).To(Equal(testExternalID), "resolved cluster external-id should equal entered value")
		})

		It("discovers ID when entered is empty and policies agree", func() {
			installerPolicy := policyWithExternalID(testExternalID)
			supportPolicy := policyWithExternalID(testExternalID)
			client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/installer").Return(roleWithPolicy(installerPolicy), nil)
			client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/support").Return(roleWithPolicy(supportPolicy), nil)

			id, err := ResolveSTSExternalIDForCluster(
				"",
				"arn:aws:iam::123:role/installer",
				"arn:aws:iam::123:role/support",
				client,
			)
			Expect(err).NotTo(HaveOccurred(), "cluster external-id discovery should succeed when policies agree")
			Expect(id).To(Equal(testExternalID), "discovered cluster external-id should match shared trust policy value")
		})

		It("reports mismatched trust policies when installer and support define different ExternalIds", func() {
			otherID := "333B9588-36A5-ECA4-BE8D-7C673B77CDCD"
			client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/installer").Return(roleWithPolicy(policyWithExternalID(testExternalID)), nil)
			client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/support").Return(roleWithPolicy(policyWithExternalID(otherID)), nil)

			result, err := ResolveSTSExternalIDForClusterDetails(
				"",
				"arn:aws:iam::123:role/installer",
				"arn:aws:iam::123:role/support",
				client,
			)
			Expect(err).NotTo(HaveOccurred(), "mismatched trust policy discovery should not return an error")
			Expect(result.ExternalID).To(BeEmpty(), "mismatched trust policies should not resolve an external-id")
			Expect(result.Ambiguous).To(BeTrue(), "mismatched trust policies should be reported as ambiguous")
			Expect(result.MismatchedTrustPolicies).To(BeTrue(),
				"mismatched trust policies should set MismatchedTrustPolicies")
		})

		It("reports ambiguous but not mismatched when both roles share multiple ExternalIds", func() {
			otherID := "333B9588-36A5-ECA4-BE8D-7C673B77CDCD"
			installerPolicy := policyWithMultipleExternalIDs([]string{testExternalID, otherID})
			supportPolicy := policyWithMultipleExternalIDs([]string{testExternalID, otherID})
			client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/installer").Return(roleWithPolicy(installerPolicy), nil)
			client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/support").Return(roleWithPolicy(supportPolicy), nil)

			result, err := ResolveSTSExternalIDForClusterDetails(
				"",
				"arn:aws:iam::123:role/installer",
				"arn:aws:iam::123:role/support",
				client,
			)
			Expect(err).NotTo(HaveOccurred(), "shared multi-value trust policy discovery should not return an error")
			Expect(result.ExternalID).To(BeEmpty(), "shared multi-value trust policies should not auto-select an external-id")
			Expect(result.Ambiguous).To(BeTrue(), "shared multi-value trust policies should be reported as ambiguous")
			Expect(result.MismatchedTrustPolicies).To(BeFalse(),
				"shared multi-value trust policies should not be reported as mismatched")
		})

		It("does not report ambiguous discovery when trust policies have no ExternalId", func() {
			policyWithoutExternalID := `{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Principal": { "AWS": "arn:aws:iam::123:role/test" },
					"Action": "sts:AssumeRole"
				}]
			}`
			client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/installer").Return(roleWithPolicy(policyWithoutExternalID), nil)
			client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/support").Return(roleWithPolicy(policyWithoutExternalID), nil)

			result, err := ResolveSTSExternalIDForClusterDetails(
				"",
				"arn:aws:iam::123:role/installer",
				"arn:aws:iam::123:role/support",
				client,
			)
			Expect(err).NotTo(HaveOccurred(), "discovery without ExternalId conditions should not return an error")
			Expect(result.ExternalID).To(BeEmpty(), "trust policies without ExternalId should not resolve an external-id")
			Expect(result.Ambiguous).To(BeFalse(), "trust policies without ExternalId should not be ambiguous")
		})

		It("does not report ambiguous discovery when a single ID is discovered", func() {
			client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/installer").Return(roleWithPolicy(policyWithExternalID(testExternalID)), nil)
			client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/support").Return(roleWithPolicy(policyWithExternalID(testExternalID)), nil)

			result, err := ResolveSTSExternalIDForClusterDetails(
				"",
				"arn:aws:iam::123:role/installer",
				"arn:aws:iam::123:role/support",
				client,
			)
			Expect(err).NotTo(HaveOccurred(), "unambiguous discovery should not return an error")
			Expect(result.ExternalID).To(Equal(testExternalID), "unambiguous discovery should return the shared external-id")
			Expect(result.Ambiguous).To(BeFalse(), "unambiguous discovery should not be marked ambiguous")
		})

		It("returns entered ID when validating explicit external-id", func() {
			client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/installer").Return(roleWithPolicy(policyWithExternalID(testExternalID)), nil)
			client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/support").Return(roleWithPolicy(policyWithExternalID(testExternalID)), nil)

			result, err := ResolveSTSExternalIDForClusterDetails(
				testExternalID,
				"arn:aws:iam::123:role/installer",
				"arn:aws:iam::123:role/support",
				client,
			)
			Expect(err).NotTo(HaveOccurred(), "explicit external-id validation should succeed")
			Expect(result.ExternalID).To(Equal(testExternalID), "result should keep entered external-id")
		})

		It("fails when entered ID is not in support trust policy", func() {
			otherID := "333B9588-36A5-ECA4-BE8D-7C673B77CDCD"
			client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/installer").Return(roleWithPolicy(policyWithExternalID(testExternalID)), nil)
			client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/support").Return(roleWithPolicy(policyWithExternalID(otherID)), nil)

			_, err := ResolveSTSExternalIDForCluster(
				testExternalID,
				"arn:aws:iam::123:role/installer",
				"arn:aws:iam::123:role/support",
				client,
			)
			Expect(err).To(HaveOccurred(), "cluster external-id validation should fail when support trust policy mismatches")
			Expect(errors.Is(err, ststrust.ErrExternalIDNotInTrustPolicy)).To(BeTrue(),
				"validation error should indicate external-id is missing from a trust policy")
		})
	})

	Describe("RoleTrustPolicyJSON", func() {
		It("fails when the role has no assume role policy document", func() {
			_, err := RoleTrustPolicyJSON(iamtypes.Role{RoleName: aws.String("missing-policy")})
			Expect(err).To(HaveOccurred(), "missing assume role policy document should fail")
		})

		It("decodes a URL-encoded assume role policy document", func() {
			policy, err := RoleTrustPolicyJSON(roleWithPolicy(policyWithExternalID(testExternalID)))
			Expect(err).NotTo(HaveOccurred(), "valid assume role policy document should decode")
			Expect(policy).To(ContainSubstring(testExternalID), "decoded policy should contain external-id")
		})
	})

	Describe("TrustPolicyJSONForRoleARN", func() {
		var (
			mockCtrl *gomock.Controller
			client   *MockClient
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			client = NewMockClient(mockCtrl)
		})

		AfterEach(func() {
			mockCtrl.Finish()
		})

		It("returns empty policy for an empty role ARN", func() {
			policy, err := TrustPolicyJSONForRoleARN(client, "")
			Expect(err).NotTo(HaveOccurred(), "empty role ARN should not error")
			Expect(policy).To(BeEmpty(), "empty role ARN should return empty policy")
		})
	})

	Describe("ValidateExistingAccountRoleExternalID", func() {
		var (
			mockCtrl *gomock.Controller
			client   *MockClient
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			client = NewMockClient(mockCtrl)
		})

		AfterEach(func() {
			mockCtrl.Finish()
		})

		It("skips validation when external-id is empty", func() {
			err := ValidateExistingAccountRoleExternalID(client, "installer-role", InstallerAccountRole, "")
			Expect(err).NotTo(HaveOccurred(), "empty external-id should skip validation")
		})

		It("skips validation for worker roles", func() {
			err := ValidateExistingAccountRoleExternalID(client, "worker-role", WorkerAccountRole, testExternalID)
			Expect(err).NotTo(HaveOccurred(), "worker roles should not require external-id validation")
		})

		It("fails when an existing installer role has no external-id", func() {
			client.EXPECT().GetRoleByName("installer-role").Return(
				roleWithPolicy(policyWithoutExternalID()), nil,
			)
			err := ValidateExistingAccountRoleExternalID(client, "installer-role", InstallerAccountRole, testExternalID)
			Expect(err).To(HaveOccurred(), "installer role without external-id should fail validation")
		})
	})

	Describe("ValidateSTSExternalIDFormat", func() {
		It("accepts an empty external-id", func() {
			Expect(ValidateSTSExternalIDFormat("")).NotTo(HaveOccurred(), "empty external-id should be valid")
		})

		It("rejects an invalid external-id", func() {
			err := ValidateSTSExternalIDFormat("x")
			Expect(err).To(HaveOccurred(), "invalid external-id format should fail validation")
		})
	})

	Describe("PreserveSTSExternalIDsInTrustPolicy edge cases", func() {
		It("returns the built policy when the existing policy has no external-id", func() {
			built := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"sts:AssumeRole","Principal":{"AWS":"arn"}}]}`
			preserved, err := PreserveSTSExternalIDsInTrustPolicy(built, policyWithoutExternalID())
			Expect(err).NotTo(HaveOccurred(), "preserving from policy without external-id should succeed")
			Expect(preserved).To(Equal(built), "built policy should remain unchanged")
		})

		It("copies StringEqualsIfExists external-id conditions", func() {
			existing := `{
				"Version":"2012-10-17",
				"Statement":[{"Effect":"Allow","Action":"sts:AssumeRole","Principal":{"AWS":"arn"},
					"Condition":{"StringEqualsIfExists":{"sts:ExternalId":"` + testExternalID + `"}}}]}
			`
			built, err := BuildAccountRoleAssumeRolePolicy(
				InstallerAccountRole, "aws", map[string]*cmv1.AWSSTSPolicy{
					"sts_installer_trust_policy": mustPolicy(installerTrustTemplate),
				}, "stage", "",
			)
			Expect(err).NotTo(HaveOccurred(), "failed to build installer trust policy template")
			preserved, err := PreserveSTSExternalIDsInTrustPolicy(built, existing)
			Expect(err).NotTo(HaveOccurred(), "preserving StringEqualsIfExists external-id should succeed")
			ids, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(preserved)
			Expect(err).NotTo(HaveOccurred(), "failed to parse preserved trust policy")
			Expect(ids).To(Equal([]string{testExternalID}), "preserved policy should contain existing external-id")
		})
	})

	Describe("GenerateAccountRolePolicyFiles", func() {
		It("generates installer trust policy files with external-id injection", func() {
			tmpDir, err := os.MkdirTemp("", "rosa-policy-*")
			Expect(err).NotTo(HaveOccurred(), "failed to create temp directory")
			wd, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred(), "failed to get working directory")
			Expect(os.Chdir(tmpDir)).To(Succeed(), "failed to change into temp directory")
			defer func() {
				Expect(os.Chdir(wd)).To(Succeed(), "failed to restore working directory")
				Expect(os.RemoveAll(tmpDir)).To(Succeed(), "failed to remove temp directory")
			}()

			policies := map[string]*cmv1.AWSSTSPolicy{
				"sts_installer_trust_policy": mustPolicy(installerTrustTemplate),
			}
			accountRoles := map[string]AccountRole{
				InstallerAccountRole: {Name: "Installer"},
			}

			err = GenerateAccountRolePolicyFiles(
				reporter.CreateReporter(), "stage", policies, true, accountRoles, "aws", testExternalID,
			)
			Expect(err).NotTo(HaveOccurred(), "policy file generation should succeed")

			policyPath := GetFormattedFileName("sts_installer_trust_policy")
			policyBytes, err := os.ReadFile(policyPath)
			Expect(err).NotTo(HaveOccurred(), "generated installer trust policy file should exist")
			ids, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(string(policyBytes))
			Expect(err).NotTo(HaveOccurred(), "generated installer trust policy should be valid JSON")
			Expect(ids).To(Equal([]string{testExternalID}),
				"generated installer trust policy file should contain the injected external-id")
		})
	})

	Describe("trust policy helper edge cases", func() {
		It("preserves external-id conditions when assume role action is an array", func() {
			existing := `{
				"Version":"2012-10-17",
				"Statement":[{"Effect":"Allow","Action":["sts:AssumeRole"],"Principal":{"AWS":"arn"},
					"Condition":{"StringEquals":{"sts:ExternalId":"` + testExternalID + `"}}}]}
			`
			built, err := BuildAccountRoleAssumeRolePolicy(
				InstallerAccountRole, "aws", map[string]*cmv1.AWSSTSPolicy{
					"sts_installer_trust_policy": mustPolicy(installerTrustTemplate),
				}, "stage", "",
			)
			Expect(err).NotTo(HaveOccurred(), "failed to build installer trust policy template")
			preserved, err := PreserveSTSExternalIDsInTrustPolicy(built, existing)
			Expect(err).NotTo(HaveOccurred(), "array action trust policy should preserve external-id")
			ids, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(preserved)
			Expect(err).NotTo(HaveOccurred(), "failed to parse preserved trust policy")
			Expect(ids).To(Equal([]string{testExternalID}), "preserved policy should contain existing external-id")
		})

		It("returns an error when the rebuilt trust policy is invalid JSON", func() {
			_, err := copySTSExternalIDConditions("{bad", policyWithExternalID(testExternalID))
			Expect(err).To(HaveOccurred(), "invalid rebuilt trust policy JSON should fail")
		})

		It("merges sts:ExternalId without discarding unrelated condition keys", func() {
			built := `{
				"Version":"2012-10-17",
				"Statement":[{"Effect":"Allow","Action":"sts:AssumeRole","Principal":{"AWS":"arn"},
					"Condition":{"StringEquals":{"aws:SourceAccount":"123456789012"}}}]
			}`
			existing := policyWithExternalID(testExternalID)
			preserved, err := copySTSExternalIDConditions(built, existing)
			Expect(err).NotTo(HaveOccurred(), "merging external-id into existing conditions should succeed")

			var doc map[string]interface{}
			Expect(json.Unmarshal([]byte(preserved), &doc)).To(Succeed(), "preserved trust policy should be valid JSON")
			statements := doc["Statement"].([]interface{})
			statement := statements[0].(map[string]interface{})
			condition := statement["Condition"].(map[string]interface{})
			stringEquals := condition["StringEquals"].(map[string]interface{})
			Expect(stringEquals["aws:SourceAccount"]).To(Equal("123456789012"),
				"unrelated condition keys should be preserved")
			Expect(stringEquals["sts:ExternalId"]).To(Equal(testExternalID),
				"sts:ExternalId should be merged into the existing condition block")
		})
	})

	Describe("BuildAccountRoleAssumeRolePolicy edge cases", func() {
		It("returns an empty policy when the template is missing but external-id is set", func() {
			policy, err := BuildAccountRoleAssumeRolePolicy(
				InstallerAccountRole, "aws", map[string]*cmv1.AWSSTSPolicy{}, "stage", testExternalID,
			)
			Expect(err).NotTo(HaveOccurred(), "missing template should not error")
			Expect(policy).To(BeEmpty(), "missing template should return empty policy")
		})
	})
})

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

// mustPolicy builds an AWSSTSPolicy from a template document or fails the test.
func mustPolicy(doc string) *cmv1.AWSSTSPolicy {
	p, err := (&cmv1.AWSSTSPolicyBuilder{}).Details(doc).Build()
	Expect(err).NotTo(HaveOccurred(), "failed to build test policy document")
	return p
}

// roleWithPolicy builds an IAM role stub with a URL-encoded assume-role policy document.
func roleWithPolicy(policyJSON string) iamtypes.Role {
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

// policyWithMultipleExternalIDs returns a trust policy JSON document with multiple sts:ExternalId values.
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
