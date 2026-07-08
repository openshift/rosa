package handler

import (
	"fmt"

	"github.com/aws/smithy-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("isAWSAuthorizationError", func() {
	It("returns true for UnauthorizedOperation", func() {
		err := &smithy.GenericAPIError{Code: "UnauthorizedOperation", Message: "not authorized"}
		Expect(isAWSAuthorizationError(err)).To(BeTrue())
	})

	It("returns true for AccessDenied", func() {
		err := &smithy.GenericAPIError{Code: "AccessDenied", Message: "access denied"}
		Expect(isAWSAuthorizationError(err)).To(BeTrue())
	})

	It("returns false for other API errors", func() {
		err := &smithy.GenericAPIError{Code: "DependencyViolation", Message: "has dependent object"}
		Expect(isAWSAuthorizationError(err)).To(BeFalse())
	})

	It("returns false for nil error", func() {
		Expect(isAWSAuthorizationError(nil)).To(BeFalse())
	})

	It("returns true for wrapped UnauthorizedOperation", func() {
		inner := &smithy.GenericAPIError{Code: "UnauthorizedOperation", Message: "ec2:DeleteSecurityGroup"}
		err := fmt.Errorf("delete security group sg-123: %w", inner)
		Expect(isAWSAuthorizationError(err)).To(BeTrue())
	})

	It("returns false for non-API errors containing auth keywords", func() {
		err := fmt.Errorf("UnauthorizedOperation: plain string, not smithy")
		Expect(isAWSAuthorizationError(err)).To(BeFalse())
	})
})

var _ = Describe("hostedZoneIsHCPInternal", func() {
	It("returns true for hypershift.local suffix", func() {
		Expect(hostedZoneIsHCPInternal("mycluster.hypershift.local")).To(BeTrue())
	})

	It("returns false for ingress zone name", func() {
		Expect(hostedZoneIsHCPInternal("rosa.mycluster.example.com")).To(BeFalse())
	})

	It("returns false for empty string", func() {
		Expect(hostedZoneIsHCPInternal("")).To(BeFalse())
	})

	It("returns false when hypershift.local appears mid-string", func() {
		Expect(hostedZoneIsHCPInternal("hypershift.local.example.com")).To(BeFalse())
	})
})
