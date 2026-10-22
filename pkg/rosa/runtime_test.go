// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package rosa

import (
	"context"
	"errors"
	"fmt"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awssts "github.com/aws/aws-sdk-go-v2/service/sts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	seaplaneclientset "github.com/openshift-online/rosa-seaplane-api/clientset"
	seaplanerest "github.com/openshift-online/rosa-seaplane-api/clientset/rest"

	reportertest "github.com/openshift/rosa/test/reporter"
)

const testSeaplaneURL = "https://abc123.execute-api.us-east-1.amazonaws.com"

var _ = Describe("WithSeaplane", func() {
	var (
		origLoadConfig  func(context.Context, string, string) (awssdk.Config, error)
		origGetIdentity func(context.Context, awssdk.Config) (*awssts.GetCallerIdentityOutput, error)
		origNewClient   func(*seaplanerest.Config) (*seaplaneclientset.Clientset, error)
		origExplicitURL func() string
		origExitFn      func(int)

		exited   bool
		errMsg   string
		fakeRept *reportertest.FakeLogger
	)

	BeforeEach(func() {
		origLoadConfig = awsLoadConfig
		origGetIdentity = awsGetIdentity
		origNewClient = seaplaneNewClient
		origExplicitURL = seaplaneExplicitURL
		origExitFn = seaplaneExitFn

		exited = false
		errMsg = ""

		seaplaneExplicitURL = func() string { return testSeaplaneURL }
		seaplaneExitFn = func(int) { exited = true }

		fakeRept = &reportertest.FakeLogger{
			ErrorFn: func(f string, a ...any) error {
				errMsg = fmt.Sprintf(f, a...)
				return nil
			},
		}
	})

	AfterEach(func() {
		awsLoadConfig = origLoadConfig
		awsGetIdentity = origGetIdentity
		seaplaneNewClient = origNewClient
		seaplaneExplicitURL = origExplicitURL
		seaplaneExitFn = origExitFn
	})

	stubLoadConfig := func() {
		awsLoadConfig = func(_ context.Context, _, _ string) (awssdk.Config, error) {
			return awssdk.Config{}, nil
		}
	}

	stubIdentity := func(accountID, arn string) {
		awsGetIdentity = func(_ context.Context, _ awssdk.Config) (*awssts.GetCallerIdentityOutput, error) {
			return &awssts.GetCallerIdentityOutput{
				Account: awssdk.String(accountID),
				Arn:     awssdk.String(arn),
				UserId:  awssdk.String("AIDAEXAMPLE"),
			}, nil
		}
	}

	It("populates Creator and SeaplaneClient on success", func() {
		GinkgoT().Setenv("AWS_REGION", "")

		stubLoadConfig()
		stubIdentity("123456789012", "arn:aws:iam::123456789012:user/test")

		var capturedCfg *seaplanerest.Config
		seaplaneNewClient = func(cfg *seaplanerest.Config) (*seaplaneclientset.Clientset, error) {
			capturedCfg = cfg
			return &seaplaneclientset.Clientset{}, nil
		}

		r := &Runtime{Reporter: fakeRept}
		r.WithSeaplane()

		Expect(exited).To(BeFalse())
		Expect(r.Creator).NotTo(BeNil())
		Expect(r.Creator.AccountID).To(Equal("123456789012"))
		Expect(r.SeaplaneClient).NotTo(BeNil())

		Expect(capturedCfg).NotTo(BeNil(), "seaplaneNewClient must have been called with a config")
		Expect(capturedCfg.Host).To(Equal(testSeaplaneURL), "Host must be the seaplane URL")
		Expect(capturedCfg.Region).To(Equal("us-east-1"), "Region must be extracted from the URL")
		Expect(capturedCfg.AccountID).To(Equal("123456789012"), "AccountID must match caller identity")
		Expect(capturedCfg.CallerARN).To(Equal("arn:aws:iam::123456789012:user/test"), "CallerARN must match caller identity")
	})

	It("exits when URL uses HTTP instead of HTTPS", func() {
		seaplaneExplicitURL = func() string { return "http://abc123.execute-api.us-east-1.amazonaws.com" }

		r := &Runtime{Reporter: fakeRept}
		r.WithSeaplane()

		Expect(exited).To(BeTrue())
		Expect(errMsg).To(ContainSubstring("must use HTTPS"))
	})

	It("exits when URL has no extractable region", func() {
		GinkgoT().Setenv("AWS_REGION", "")
		GinkgoT().Setenv("AWS_DEFAULT_REGION", "")

		seaplaneExplicitURL = func() string { return "https://example.com/api" }

		r := &Runtime{Reporter: fakeRept}
		r.WithSeaplane()

		Expect(exited).To(BeTrue())
		Expect(errMsg).To(ContainSubstring("cannot derive AWS region"))
	})

	It("exits when explicit region conflicts with URL region", func() {
		GinkgoT().Setenv("AWS_REGION", "us-west-2")

		r := &Runtime{Reporter: fakeRept}
		r.WithSeaplane()

		Expect(exited).To(BeTrue())
		Expect(errMsg).To(ContainSubstring("us-west-2"))
		Expect(errMsg).To(ContainSubstring("us-east-1"))
	})

	It("exits when CreatorForCallerIdentity fails due to bad ARN", func() {
		stubLoadConfig()
		awsGetIdentity = func(_ context.Context, _ awssdk.Config) (*awssts.GetCallerIdentityOutput, error) {
			return &awssts.GetCallerIdentityOutput{
				Account: awssdk.String("123456789012"),
				Arn:     awssdk.String("not-a-valid-arn"),
				UserId:  awssdk.String("AIDAEXAMPLE"),
			}, nil
		}

		r := &Runtime{Reporter: fakeRept}
		r.WithSeaplane()

		Expect(exited).To(BeTrue())
		Expect(errMsg).To(ContainSubstring("Failed to build creator from caller identity"))
	})

	It("exits when LoadDefaultConfig fails", func() {
		awsLoadConfig = func(_ context.Context, _, _ string) (awssdk.Config, error) {
			return awssdk.Config{}, errors.New("no credentials")
		}

		r := &Runtime{Reporter: fakeRept}
		r.WithSeaplane()

		Expect(exited).To(BeTrue())
		Expect(errMsg).To(ContainSubstring("Failed to load AWS config"))
	})

	It("exits when GetCallerIdentity fails", func() {
		stubLoadConfig()
		awsGetIdentity = func(_ context.Context, _ awssdk.Config) (*awssts.GetCallerIdentityOutput, error) {
			return nil, errors.New("sts error")
		}

		r := &Runtime{Reporter: fakeRept}
		r.WithSeaplane()

		Expect(exited).To(BeTrue())
		Expect(errMsg).To(ContainSubstring("Failed to get AWS caller identity"))
	})

	It("exits when NewForConfig fails", func() {
		stubLoadConfig()
		stubIdentity("123456789012", "arn:aws:iam::123456789012:user/test")
		seaplaneNewClient = func(_ *seaplanerest.Config) (*seaplaneclientset.Clientset, error) {
			return nil, errors.New("client error")
		}

		r := &Runtime{Reporter: fakeRept}
		r.WithSeaplane()

		Expect(exited).To(BeTrue())
		Expect(errMsg).To(ContainSubstring("Failed to build Platform API client"))
	})
})
