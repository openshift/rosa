package handler

import (
	"context"
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
