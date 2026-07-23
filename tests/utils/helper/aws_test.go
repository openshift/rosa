package helper

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"
)

var _ = Describe("isAWSCredentialError", func() {
	DescribeTable("recognizes credential failures",
		func(errMsg string, expected bool) {
			Expect(isAWSCredentialError(errMsg)).To(Equal(expected))
		},
		Entry("InvalidClientTokenId",
			"operation error STS: GetCallerIdentity, https response error StatusCode: 403, "+
				"RequestID: abc, api error InvalidClientTokenId: The security token included in the request is invalid",
			true),
		Entry("ExpiredToken",
			"ExpiredToken: The security token included in the request is expired",
			true),
		Entry("RequestExpired",
			"RequestExpired: Request has expired",
			true),
		Entry("OCM wrapped AWS creds",
			"AWS was not able to validate the provided access credentials",
			true),
		Entry("NoCredentialProviders",
			"NoCredentialProviders: no valid providers in chain",
			true),
		Entry("refresh failure",
			"failed to refresh cached credentials, expired",
			true),
		Entry("unrelated error",
			"the subnet ID 'subnet-xxx' does not exist",
			false),
		Entry("empty",
			"",
			false),
	)
})

var _ = Describe("AWSCredentialsInvalid", func() {
	var orig func() (*aws_client.AWSClient, error)

	BeforeEach(func() {
		orig = createAWSClientForCredCheck
	})

	AfterEach(func() {
		createAWSClientForCredCheck = orig
	})

	It("returns false when credentials are valid", func() {
		createAWSClientForCredCheck = func() (*aws_client.AWSClient, error) {
			return &aws_client.AWSClient{}, nil
		}
		invalid, err := AWSCredentialsInvalid()
		Expect(invalid).To(BeFalse())
		Expect(err).ToNot(HaveOccurred())
	})

	It("returns true when credentials are invalid", func() {
		createAWSClientForCredCheck = func() (*aws_client.AWSClient, error) {
			return nil, fmt.Errorf("api error InvalidClientTokenId: The security token included in the request is invalid")
		}
		invalid, err := AWSCredentialsInvalid()
		Expect(invalid).To(BeTrue())
		Expect(err).To(HaveOccurred())
	})

	It("returns false for unrelated create errors", func() {
		createAWSClientForCredCheck = func() (*aws_client.AWSClient, error) {
			return nil, fmt.Errorf("dial tcp: lookup sts.amazonaws.com: no such host")
		}
		invalid, err := AWSCredentialsInvalid()
		Expect(invalid).To(BeFalse())
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("SkipIfAWSCredentialsInvalid", func() {
	var (
		origClient func() (*aws_client.AWSClient, error)
		origSkip   func(message string, callerSkip ...int)
		origDoc    func(text string, callback ...func())
	)

	BeforeEach(func() {
		origClient = createAWSClientForCredCheck
		origSkip = skipForInfra
		origDoc = documentStep
		documentStep = func(_ string, _ ...func()) {}
	})

	AfterEach(func() {
		createAWSClientForCredCheck = origClient
		skipForInfra = origSkip
		documentStep = origDoc
	})

	It("skips on invalid credentials", func() {
		createAWSClientForCredCheck = func() (*aws_client.AWSClient, error) {
			return nil, fmt.Errorf("api error InvalidClientTokenId: The security token included in the request is invalid")
		}
		var skippedMsg string
		skipForInfra = func(message string, _ ...int) {
			skippedMsg = message
		}
		SkipIfAWSCredentialsInvalid()
		Expect(skippedMsg).ToNot(BeEmpty())
		Expect(skippedMsg).To(ContainSubstring("infra, not product"))
	})

	It("does not skip when credentials are valid", func() {
		createAWSClientForCredCheck = func() (*aws_client.AWSClient, error) {
			return &aws_client.AWSClient{}, nil
		}
		called := false
		skipForInfra = func(message string, _ ...int) {
			called = true
		}
		SkipIfAWSCredentialsInvalid()
		Expect(called).To(BeFalse())
	})
})
