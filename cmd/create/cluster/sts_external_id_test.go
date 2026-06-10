package cluster

import (
	"net/url"

	"go.uber.org/mock/gomock"

	"github.com/aws/aws-sdk-go-v2/aws"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pkgaws "github.com/openshift/rosa/pkg/aws"
)

const testExternalID = "223B9588-36A5-ECA4-BE8D-7C673B77CEC1"

var _ = Describe("STS external ID cluster helpers", func() {
	var (
		mockCtrl *gomock.Controller
		client   *pkgaws.MockClient
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		client = pkgaws.NewMockClient(mockCtrl)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("validateChangedSTSExternalIDFlag accepts a valid external-id", func() {
		err := validateChangedSTSExternalIDFlag(testExternalID)
		Expect(err).NotTo(HaveOccurred(), "valid external-id should pass format validation")
	})

	It("validateChangedSTSExternalIDFlag rejects an invalid external-id", func() {
		err := validateChangedSTSExternalIDFlag("x")
		Expect(err).To(HaveOccurred(), "invalid external-id should fail format validation")
	})

	It("resolveSTSExternalIDForClusterCreate returns entered external-id without role ARNs", func() {
		result, err := resolveSTSExternalIDForClusterCreate(client, testExternalID, "", "")
		Expect(err).NotTo(HaveOccurred(), "resolution without role ARNs should succeed")
		Expect(result.ExternalID).To(Equal(testExternalID), "entered external-id should be preserved")
	})

	It("resolveSTSExternalIDForClusterCreate resolves trust policies for both roles", func() {
		installerPolicy := policyWithExternalID(testExternalID)
		supportPolicy := policyWithExternalID(testExternalID)
		client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/installer").Return(roleWithPolicy(installerPolicy), nil)
		client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/support").Return(roleWithPolicy(supportPolicy), nil)

		result, err := resolveSTSExternalIDForClusterCreate(
			client, testExternalID, "arn:aws:iam::123:role/installer", "arn:aws:iam::123:role/support",
		)
		Expect(err).NotTo(HaveOccurred(), "matching trust policies should resolve external-id")
		Expect(result.ExternalID).To(Equal(testExternalID), "resolved external-id should match entered value")
	})

	It("checkSTSExternalIDResolution fails on mismatched trust policies", func() {
		err := checkSTSExternalIDResolution(pkgaws.STSExternalIDClusterResolution{
			MismatchedTrustPolicies: true,
		}, false)
		Expect(err).To(MatchError(errMismatchedSTSExternalIDTrustPolicies),
			"mismatched trust policies should fail when external-id flag is unset")
	})

	It("shouldWarnAmbiguousSTSExternalID reports ambiguous discovery", func() {
		Expect(shouldWarnAmbiguousSTSExternalID(pkgaws.STSExternalIDClusterResolution{
			Ambiguous: true,
		}, false)).To(BeTrue(), "ambiguous discovery should warn when external-id flag is unset")
	})

	It("resolveEnteredSTSExternalIDForCluster validates and resolves entered external-id", func() {
		installerPolicy := policyWithExternalID(testExternalID)
		supportPolicy := policyWithExternalID(testExternalID)
		client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/installer").Return(roleWithPolicy(installerPolicy), nil)
		client.EXPECT().GetRoleByARN("arn:aws:iam::123:role/support").Return(roleWithPolicy(supportPolicy), nil)

		externalID, err := resolveEnteredSTSExternalIDForCluster(
			client, testExternalID, "arn:aws:iam::123:role/installer", "arn:aws:iam::123:role/support",
		)
		Expect(err).NotTo(HaveOccurred(), "entered external-id should resolve against both roles")
		Expect(externalID).To(Equal(testExternalID), "resolved external-id should match entered value")
	})

	It("resolveEnteredSTSExternalIDForCluster rejects an invalid entered external-id", func() {
		_, err := resolveEnteredSTSExternalIDForCluster(
			client, "x", "arn:aws:iam::123:role/installer", "arn:aws:iam::123:role/support",
		)
		Expect(err).To(HaveOccurred(), "invalid entered external-id should fail validation")
	})
})

func roleWithPolicy(policyJSON string) iamtypes.Role {
	encoded := url.QueryEscape(policyJSON)
	return iamtypes.Role{
		RoleName:                 aws.String("test-role"),
		AssumeRolePolicyDocument: aws.String(encoded),
	}
}

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
