package config

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func awsRetryBuffer(content string) bytes.Buffer {
	var out bytes.Buffer
	out.WriteString(content)
	return out
}

var _ = Describe("isAWSRegionsCredentialTransient", func() {
	DescribeTable("classifies AWS regions and credential flakes",
		func(outContent string, err error, expected bool) {
			Expect(isAWSRegionsCredentialTransient(awsRetryBuffer(outContent), err)).To(Equal(expected))
		},
		Entry("empty output and no error", "", nil, false),
		Entry("regions failure in output", "Failed to retrieve AWS regions", nil, true),
		Entry("regions failure in error", "", errors.New("Failed to retrieve AWS regions"), true),
		Entry("credential validation failure in output",
			"AWS was not able to validate the provided access credentials", nil, true),
		Entry("credential validation failure in error", "",
			errors.New("AWS was not able to validate the provided access credentials"), true),
		Entry("validation error unrelated to regions or credentials",
			"ERR: expected valid allowed registries for import values",
			errors.New("expected valid allowed registries for import values"), false),
		Entry("generic CLI failure", "command failed", errors.New("exit status 1"), false),
	)
})

var _ = Describe("RetryOnAWSRegionsCredentialError", func() {
	const retryDelay = time.Millisecond

	It("returns immediately when fn succeeds", func() {
		calls := 0
		expected := awsRetryBuffer("ok")

		out, err := RetryOnAWSRegionsCredentialError(func() (bytes.Buffer, error) {
			calls++
			return expected, nil
		}, 3, retryDelay)

		Expect(err).ToNot(HaveOccurred())
		Expect(out.String()).To(Equal("ok"))
		Expect(calls).To(Equal(1))
	})

	It("returns immediately when fn returns a non-transient error", func() {
		calls := 0
		expectedErr := errors.New("ERR: invalid subnet")

		out, err := RetryOnAWSRegionsCredentialError(func() (bytes.Buffer, error) {
			calls++
			return awsRetryBuffer("validation failed"), expectedErr
		}, 3, retryDelay)

		Expect(err).To(MatchError(expectedErr))
		Expect(out.String()).To(Equal("validation failed"))
		Expect(calls).To(Equal(1))
	})

	It("retries transient failures until fn succeeds", func() {
		calls := 0
		transientErr := errors.New("Failed to retrieve AWS regions")

		out, err := RetryOnAWSRegionsCredentialError(func() (bytes.Buffer, error) {
			calls++
			if calls < 3 {
				return awsRetryBuffer("transient"), transientErr
			}
			return awsRetryBuffer("expected validation error"), errors.New("expected validation error")
		}, 3, retryDelay)

		Expect(err).To(MatchError(errors.New("expected validation error")))
		Expect(out.String()).To(Equal("expected validation error"))
		Expect(calls).To(Equal(3))
	})

	It("stops after maxRetries and returns the last transient result", func() {
		calls := 0
		transientErr := errors.New("AWS was not able to validate the provided access credentials")

		out, err := RetryOnAWSRegionsCredentialError(func() (bytes.Buffer, error) {
			calls++
			return awsRetryBuffer(fmt.Sprintf("attempt-%d", calls)), transientErr
		}, 2, retryDelay)

		Expect(err).To(MatchError(transientErr))
		Expect(out.String()).To(Equal("attempt-3"))
		Expect(calls).To(Equal(3))
	})

	It("does not retry when maxRetries is zero", func() {
		calls := 0
		transientErr := errors.New("Failed to retrieve AWS regions")

		out, err := RetryOnAWSRegionsCredentialError(func() (bytes.Buffer, error) {
			calls++
			return awsRetryBuffer("only-once"), transientErr
		}, 0, retryDelay)

		Expect(err).To(MatchError(transientErr))
		Expect(out.String()).To(Equal("only-once"))
		Expect(calls).To(Equal(1))
	})
})
