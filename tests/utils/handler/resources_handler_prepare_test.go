// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	iamTypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/smithy-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/seaplane"
	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/utils/helper"
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
		helper.ExpectErrorWithMessage(err, "describing shared subnets")
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

var _ = Describe("oidcConfigIDFromCreateOutput", func() {
	DescribeTable("selects the resource identifier from the create response",
		func(responseJSON, expected string) {
			var response bytes.Buffer
			_, err := response.WriteString(responseJSON)
			Expect(err).NotTo(HaveOccurred())
			Expect(oidcConfigIDFromCreateOutput(response)).To(Equal(expected))
		},
		Entry("legacy OCM id", `{"id":"legacy-id","metadata":{"name":"legacy-name"}}`, "legacy-id"),
		Entry("Platform API UID", `{"metadata":{"name":"managed-name","uid":"platform-uid"}}`, "platform-uid"),
		Entry("name-only compatibility response", `{"metadata":{"name":"legacy-name"}}`, "legacy-name"),
	)
})

var _ = Describe("PrepareAdditionalPrincipalsRole", func() {
	AfterEach(func() {
		seaplane.Reset()
	})

	It("requires an installer role ARN before creating AWS resources", func() {
		seaplane.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
		rh := &resourcesHandler{resources: &Resources{}}
		roleArn, err := rh.PrepareAdditionalPrincipalsRole("additional-role", "")
		Expect(err).To(MatchError("installer role ARN is required to create an additional principal role"))
		Expect(roleArn).To(BeEmpty())
	})
})

var _ = Describe("waitForAdditionalPrincipalRole", func() {
	It("adapts the AWS role response to its ARN", func() {
		creator := newAdditionalPrincipalRoleCreator(
			func(roleName string, trustedRoleArn string) (iamTypes.Role, error) {
				Expect(roleName).To(Equal("additional-role"))
				Expect(trustedRoleArn).To(Equal("arn:aws:iam::123456789012:role/control-plane-operator"))
				return iamTypes.Role{Arn: aws.String("arn:aws:iam::123456789012:role/additional-role")}, nil
			},
			"additional-role",
			"arn:aws:iam::123456789012:role/control-plane-operator",
		)
		roleArn, err := creator()
		Expect(err).NotTo(HaveOccurred())
		Expect(roleArn).To(Equal("arn:aws:iam::123456789012:role/additional-role"))
	})

	It("retries while the trusted role principal propagates", func() {
		var calls int
		roleArn, err := waitForAdditionalPrincipalRole(context.Background(), time.Millisecond, func() (string, error) {
			calls++
			if calls == 1 {
				return "", &smithy.GenericAPIError{
					Code:    "MalformedPolicyDocument",
					Message: "Invalid principal in policy",
				}
			}
			return "arn:aws:iam::123456789012:role/additional-role", nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(roleArn).To(Equal("arn:aws:iam::123456789012:role/additional-role"))
		Expect(calls).To(Equal(2))
	})

	It("does not retry other IAM failures", func() {
		var calls int
		_, err := waitForAdditionalPrincipalRole(context.Background(), time.Millisecond, func() (string, error) {
			calls++
			return "", &smithy.GenericAPIError{Code: "AccessDenied", Message: "not authorized"}
		})
		Expect(err).To(HaveOccurred())
		Expect(calls).To(Equal(1))
	})
})

var _ = Describe("NewClusterHandlerForKonflux", func() {
	AfterEach(func() {
		seaplane.Reset()
	})

	It("does not require shared account credentials for Seaplane profiles", func() {
		seaplane.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
		tempDir := GinkgoT().TempDir()
		konfluxResources := filepath.Join(tempDir, "konflux-resources.json")
		Expect(os.WriteFile(
			konfluxResources,
			[]byte(`{"additional_principals":"arn:aws:iam::123456789012:role/test-role"}`),
			0o600,
		)).To(Succeed())

		originalFile := config.Test.KonfluxUserDataFile
		config.Test.KonfluxUserDataFile = konfluxResources
		DeferCleanup(func() {
			config.Test.KonfluxUserDataFile = originalFile
		})

		_, err := NewClusterHandlerForKonflux(nil, &Profile{
			Region:        "us-east-1",
			ClusterConfig: &ClusterConfig{AdditionalPrincipals: true},
		})
		Expect(err).NotTo(HaveOccurred())
	})
})
