package handler

import (
	"context"
	"fmt"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("waitForSubnetsVisible", func() {
	It("returns nil immediately for empty subnet list", func() {
		never := func(_ context.Context, _ []string) (int, error) {
			Fail("checker should not be called for empty list")
			return 0, nil
		}
		Expect(waitForSubnetsVisible(nil, never)).To(Succeed())
		Expect(waitForSubnetsVisible([]string{}, never)).To(Succeed())
	})

	It("succeeds when all subnets are found on first call", func() {
		checker := func(_ context.Context, ids []string) (int, error) {
			return len(ids), nil
		}
		Expect(waitForSubnetsVisible([]string{"subnet-aaa", "subnet-bbb"}, checker)).To(Succeed())
	})

	It("retries on InvalidSubnetID.NotFound then succeeds", func() {
		var calls int32
		checker := func(_ context.Context, ids []string) (int, error) {
			n := atomic.AddInt32(&calls, 1)
			if n <= 2 {
				return 0, fmt.Errorf("InvalidSubnetID.NotFound: subnet-aaa")
			}
			return len(ids), nil
		}
		Expect(waitForSubnetsVisible([]string{"subnet-aaa"}, checker)).To(Succeed())
		Expect(atomic.LoadInt32(&calls)).To(BeNumerically(">=", 3))
	})

	It("returns error for non-retryable failures", func() {
		checker := func(_ context.Context, _ []string) (int, error) {
			return 0, fmt.Errorf("InternalError: service unavailable")
		}
		err := waitForSubnetsVisible([]string{"subnet-aaa"}, checker)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("describing shared subnets"))
	})
})

var _ = Describe("extractSubnetIDsFromArns", func() {
	It("extracts subnet IDs from mixed ARN list", func() {
		arns := []string{
			"arn:aws:ec2:us-east-1:123456789012:subnet/subnet-aaa111",
			"arn:aws:ec2:us-east-1:123456789012:security-group/sg-bbb222",
			"arn:aws:ec2:us-east-1:123456789012:subnet/subnet-ccc333",
		}
		ids := extractSubnetIDsFromArns(arns)
		Expect(ids).To(Equal([]string{"subnet-aaa111", "subnet-ccc333"}))
	})

	It("returns nil for empty list", func() {
		Expect(extractSubnetIDsFromArns(nil)).To(BeNil())
		Expect(extractSubnetIDsFromArns([]string{})).To(BeNil())
	})

	It("returns nil when no subnet ARNs present", func() {
		arns := []string{
			"arn:aws:ec2:us-east-1:123456789012:security-group/sg-bbb222",
		}
		Expect(extractSubnetIDsFromArns(arns)).To(BeNil())
	})
})
