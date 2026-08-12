package rosa

import (
	"context"
	"errors"
	"fmt"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awssts "github.com/aws/aws-sdk-go-v2/service/sts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	hfrest "github.com/openshift-online/rosa-hyperfleet-api/clientset/rest"

	reportertest "github.com/openshift/rosa/test/reporter"
)

const testHyperfleetURL = "https://abc123.execute-api.us-east-1.amazonaws.com"

var _ = Describe("WithHyperFleet", func() {
	var (
		origLoadConfig  func(context.Context, string, string) (awssdk.Config, error)
		origGetIdentity func(context.Context, awssdk.Config) (*awssts.GetCallerIdentityOutput, error)
		origNewClient   func(*hfrest.Config) (*hyperfleetclientset.Clientset, error)
		origExplicitURL func() string
		origExitFn      func(int)

		exited   bool
		errMsg   string
		fakeRept *reportertest.FakeLogger
	)

	BeforeEach(func() {
		origLoadConfig = awsLoadConfig
		origGetIdentity = awsGetIdentity
		origNewClient = hfNewClient
		origExplicitURL = hfExplicitURL
		origExitFn = hfExitFn

		exited = false
		errMsg = ""

		hfExplicitURL = func() string { return testHyperfleetURL }
		hfExitFn = func(int) { exited = true }

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
		hfNewClient = origNewClient
		hfExplicitURL = origExplicitURL
		hfExitFn = origExitFn
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

	It("populates Creator and HyperFleetClient on success", func() {
		GinkgoT().Setenv("AWS_REGION", "")

		stubLoadConfig()
		stubIdentity("123456789012", "arn:aws:iam::123456789012:user/test")

		var capturedCfg *hfrest.Config
		hfNewClient = func(cfg *hfrest.Config) (*hyperfleetclientset.Clientset, error) {
			capturedCfg = cfg
			return &hyperfleetclientset.Clientset{}, nil
		}

		r := &Runtime{Reporter: fakeRept}
		r.WithHyperFleet()

		Expect(exited).To(BeFalse())
		Expect(r.Creator).NotTo(BeNil())
		Expect(r.Creator.AccountID).To(Equal("123456789012"))
		Expect(r.HyperFleetClient).NotTo(BeNil())

		Expect(capturedCfg).NotTo(BeNil(), "hfNewClient must have been called with a config")
		Expect(capturedCfg.Host).To(Equal(testHyperfleetURL), "Host must be the hyperfleet URL")
		Expect(capturedCfg.Region).To(Equal("us-east-1"), "Region must be extracted from the URL")
		Expect(capturedCfg.AccountID).To(Equal("123456789012"), "AccountID must match caller identity")
		Expect(capturedCfg.CallerARN).To(Equal("arn:aws:iam::123456789012:user/test"), "CallerARN must match caller identity")
	})

	It("exits when URL uses HTTP instead of HTTPS", func() {
		hfExplicitURL = func() string { return "http://abc123.execute-api.us-east-1.amazonaws.com" }

		r := &Runtime{Reporter: fakeRept}
		r.WithHyperFleet()

		Expect(exited).To(BeTrue())
		Expect(errMsg).To(ContainSubstring("must use HTTPS"))
	})

	It("exits when URL has no extractable region", func() {
		hfExplicitURL = func() string { return "https://example.com/api" }

		r := &Runtime{Reporter: fakeRept}
		r.WithHyperFleet()

		Expect(exited).To(BeTrue())
		Expect(errMsg).To(ContainSubstring("cannot derive AWS region"))
	})

	It("exits when explicit region conflicts with URL region", func() {
		GinkgoT().Setenv("AWS_REGION", "us-west-2")

		r := &Runtime{Reporter: fakeRept}
		r.WithHyperFleet()

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
		r.WithHyperFleet()

		Expect(exited).To(BeTrue())
		Expect(errMsg).To(ContainSubstring("Failed to build creator from caller identity"))
	})

	It("exits when LoadDefaultConfig fails", func() {
		awsLoadConfig = func(_ context.Context, _, _ string) (awssdk.Config, error) {
			return awssdk.Config{}, errors.New("no credentials")
		}

		r := &Runtime{Reporter: fakeRept}
		r.WithHyperFleet()

		Expect(exited).To(BeTrue())
		Expect(errMsg).To(ContainSubstring("Failed to load AWS config"))
	})

	It("exits when GetCallerIdentity fails", func() {
		stubLoadConfig()
		awsGetIdentity = func(_ context.Context, _ awssdk.Config) (*awssts.GetCallerIdentityOutput, error) {
			return nil, errors.New("sts error")
		}

		r := &Runtime{Reporter: fakeRept}
		r.WithHyperFleet()

		Expect(exited).To(BeTrue())
		Expect(errMsg).To(ContainSubstring("Failed to get AWS caller identity"))
	})

	It("exits when NewForConfig fails", func() {
		stubLoadConfig()
		stubIdentity("123456789012", "arn:aws:iam::123456789012:user/test")
		hfNewClient = func(_ *hfrest.Config) (*hyperfleetclientset.Clientset, error) {
			return nil, errors.New("client error")
		}

		r := &Runtime{Reporter: fakeRept}
		r.WithHyperFleet()

		Expect(exited).To(BeTrue())
		Expect(errMsg).To(ContainSubstring("Failed to build Platform API client"))
	})
})
