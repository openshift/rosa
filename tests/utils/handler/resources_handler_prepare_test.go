package handler

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/aws/smithy-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/tests/ci/config"
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
		hyperfleet.Reset()
	})

	It("requires an installer role ARN before creating AWS resources", func() {
		hyperfleet.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
		rh := &resourcesHandler{resources: &Resources{}}
		roleArn, err := rh.PrepareAdditionalPrincipalsRole("additional-role", "")
		Expect(err).To(MatchError("installer role ARN is required to create an additional principal role"))
		Expect(roleArn).To(BeEmpty())
	})
})

var _ = Describe("NewClusterHandlerForKonflux", func() {
	AfterEach(func() {
		hyperfleet.Reset()
	})

	It("does not require shared account credentials for HyperFleet profiles", func() {
		hyperfleet.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
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
