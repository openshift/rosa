package hyperfleet

import (
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Platform API error details", func() {
	It("includes rejected fields and preserves the underlying error", func() {
		apiErr := &apierrors.StatusError{ErrStatus: metav1.Status{
			Message: "CLUSTERS-MGMT-VALIDATION-001: validation failed",
			Details: &metav1.StatusDetails{Causes: []metav1.StatusCause{
				{Field: "spec.hostedCluster.configuration.proxy.noProxy", Message: "field is platform-managed"},
				{Field: "spec.hostedCluster.configuration.proxy", Message: "field cannot be set"},
			}},
		}}
		result := WithAPIErrorDetails(fmt.Errorf("update failed: %w", apiErr))
		Expect(result).To(MatchError(ContainSubstring(
			"spec.hostedCluster.configuration.proxy.noProxy: field is platform-managed")))
		Expect(result).To(MatchError(ContainSubstring(
			"spec.hostedCluster.configuration.proxy: field cannot be set")))
		Expect(errors.Is(result, apiErr)).To(BeTrue())
	})

	It("preserves errors without structured details", func() {
		err := errors.New("connection failed")
		Expect(WithAPIErrorDetails(err)).To(BeIdenticalTo(err))
		apiErr := &apierrors.StatusError{ErrStatus: metav1.Status{Message: "validation failed"}}
		Expect(WithAPIErrorDetails(apiErr)).To(BeIdenticalTo(apiErr))
		Expect(WithAPIErrorDetails(nil)).To(BeNil())
	})

	It("includes causes without a field name", func() {
		apiErr := &apierrors.StatusError{ErrStatus: metav1.Status{
			Message: "validation failed",
			Details: &metav1.StatusDetails{Causes: []metav1.StatusCause{{Message: "unsupported update"}}},
		}}
		Expect(WithAPIErrorDetails(apiErr)).To(MatchError("validation failed (unsupported update)"))
	})
})
