package errors

import (
	"errors"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestErrors(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "errors testing")
}

var _ = Describe("ValidationError", func() {
	Context("as an aggregate wrapper (Err only)", func() {
		It("preserves the wrapped error's text verbatim", func() {
			inner := fmt.Errorf("cluster name is required")
			validationErr := &ValidationError{Err: inner}

			Expect(validationErr.Error()).To(Equal("cluster name is required"))
		})

		It("unwraps to the wrapped error", func() {
			inner := fmt.Errorf("cluster name is required")
			validationErr := &ValidationError{Err: inner}

			Expect(errors.Unwrap(validationErr)).To(Equal(inner))
		})

		It("is detectable with errors.As through an outer wrap", func() {
			inner := fmt.Errorf("invalid request: %s", "cluster name is required")
			wrapped := fmt.Errorf("create failed: %w", &ValidationError{Err: inner})

			var validationErr *ValidationError
			Expect(errors.As(wrapped, &validationErr)).To(BeTrue())
			Expect(validationErr.Error()).To(Equal(inner.Error()))
		})

		It("does not match a plain error via errors.As", func() {
			plain := fmt.Errorf("failed to create role: network error")

			var validationErr *ValidationError
			Expect(errors.As(plain, &validationErr)).To(BeFalse())
		})

		It("preserves errors.Is against a sentinel wrapped further down the chain", func() {
			sentinel := errors.New("boom")
			wrapped := fmt.Errorf("invalid request: %w", sentinel)
			validationErr := &ValidationError{Err: wrapped}

			Expect(errors.Is(validationErr, sentinel)).To(BeTrue())
		})
	})

	Context("as a leaf-level, field-carrying error", func() {
		It("renders only the message, never the field, so introducing Field never changes user-facing text", func() {
			validationErr := &ValidationError{Field: "ClusterName", Message: "cluster name is required"}

			Expect(validationErr.Error()).To(Equal("cluster name is required"))
		})

		It("exposes Field for structured consumers without needing to parse the message", func() {
			validationErr := &ValidationError{Field: "RoleName", Message: "role name must not be blank when provided"}

			Expect(validationErr.Field).To(Equal("RoleName"))
		})

		It("appends a wrapped cause to the message and preserves it for errors.Is", func() {
			cause := errors.New("must be a valid DNS-1123 label")
			validationErr := &ValidationError{
				Field:   "ServiceAccounts",
				Message: `invalid service account name "BAD"`,
				Err:     cause,
			}

			Expect(validationErr.Error()).To(Equal(`invalid service account name "BAD": must be a valid DNS-1123 label`))
			Expect(errors.Is(validationErr, cause)).To(BeTrue())
		})

		It("is found by errors.As even nested inside an errors.Join tree with no outer wrapper", func() {
			leaf := &ValidationError{Field: "ClusterName", Message: "cluster name is required"}
			joined := errors.Join(fmt.Errorf("some other failure"), leaf)
			wrapped := fmt.Errorf("invalid request: %w", joined)

			var found *ValidationError
			Expect(errors.As(wrapped, &found)).To(BeTrue())
			Expect(found.Field).To(Equal("ClusterName"))
		})
	})

	Context("when constructed without a Message or Err", func() {
		It("falls back to a non-empty message instead of an empty string", func() {
			validationErr := &ValidationError{}

			Expect(validationErr.Error()).To(Equal("validation failed"))
		})

		It("includes the field in the fallback message when Field is set", func() {
			validationErr := &ValidationError{Field: "ClusterName"}

			Expect(validationErr.Error()).To(Equal("validation failed: ClusterName"))
		})
	})
})
