package handler

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"github.com/aws/smithy-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("waitForSubnetsVisible", func() {
	It("returns nil immediately for empty subnet list", func() {
		never := func(_ context.Context, _ []string) (int, error) {
			Fail("checker should not be called for empty list")
			return 0, nil
		}
		Expect(waitForSubnetsVisible(context.TODO(), nil, never)).To(Succeed())
		Expect(waitForSubnetsVisible(context.TODO(), []string{}, never)).To(Succeed())
	})

	It("succeeds when all subnets are found on first call", func() {
		checker := func(_ context.Context, ids []string) (int, error) {
			return len(ids), nil
		}
		Expect(waitForSubnetsVisible(context.TODO(), []string{"subnet-aaa", "subnet-bbb"}, checker)).To(Succeed())
	})

	It("retries on InvalidSubnetID.NotFound then succeeds", func() {
		var calls int32
		checker := func(_ context.Context, ids []string) (int, error) {
			n := atomic.AddInt32(&calls, 1)
			if n <= 2 {
				return 0, &smithy.GenericAPIError{
					Code:    "InvalidSubnetID.NotFound",
					Message: "subnet-aaa does not exist",
				}
			}
			return len(ids), nil
		}
		Expect(waitForSubnetsVisible(context.TODO(), []string{"subnet-aaa"}, checker)).To(Succeed())
		Expect(atomic.LoadInt32(&calls)).To(BeNumerically(">=", 3))
	})

	It("returns error for non-retryable failures", func() {
		checker := func(_ context.Context, _ []string) (int, error) {
			return 0, &smithy.GenericAPIError{Code: "InternalError", Message: "service unavailable"}
		}
		err := waitForSubnetsVisible(context.TODO(), []string{"subnet-aaa"}, checker)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("describing shared subnets"))
	})

	It("respects context cancellation", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		checker := func(_ context.Context, _ []string) (int, error) {
			return 0, &smithy.GenericAPIError{
				Code:    "InvalidSubnetID.NotFound",
				Message: "not yet",
			}
		}
		err := waitForSubnetsVisible(ctx, []string{"subnet-aaa"}, checker)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("platformAPIOperatorTrustPolicy", func() {
	It("scopes image registry role trust to its issuer, service accounts, and audience", func() {
		const issuer = "oidc.example.com/us-east-1/issuer-id"
		policyJSON := platformAPIOperatorTrustPolicy("aws", "123456789012", issuer, []platformAPIServiceAccount{
			{"openshift-image-registry", "cluster-image-registry-operator"},
			{"openshift-image-registry", "registry"},
		})

		var policy struct {
			Statement []struct {
				Principal struct {
					Federated string `json:"Federated"`
				} `json:"Principal"`
				Action    string `json:"Action"`
				Condition struct {
					StringEquals map[string]json.RawMessage `json:"StringEquals"`
				} `json:"Condition"`
			} `json:"Statement"`
		}
		Expect(json.Unmarshal([]byte(policyJSON), &policy)).To(Succeed())
		Expect(policy.Statement).To(HaveLen(1))

		statement := policy.Statement[0]
		Expect(statement.Principal.Federated).To(Equal("arn:aws:iam::123456789012:oidc-provider/" + issuer))
		Expect(statement.Action).To(Equal("sts:AssumeRoleWithWebIdentity"))
		Expect(statement.Condition.StringEquals).To(HaveLen(2))

		var subjects []string
		Expect(json.Unmarshal(statement.Condition.StringEquals[issuer+":sub"], &subjects)).To(Succeed())
		Expect(subjects).To(ConsistOf(
			"system:serviceaccount:openshift-image-registry:cluster-image-registry-operator",
			"system:serviceaccount:openshift-image-registry:registry",
		))

		var audience string
		Expect(json.Unmarshal(statement.Condition.StringEquals[issuer+":aud"], &audience)).To(Succeed())
		Expect(audience).To(Equal("openshift"))
	})
})
