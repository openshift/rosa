package rosa

import (
	"context"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awssts "github.com/aws/aws-sdk-go-v2/service/sts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	hfrest "github.com/openshift-online/rosa-hyperfleet-api/clientset/rest"
	"github.com/spf13/cobra"

	reportertest "github.com/openshift/rosa/test/reporter"
)

func TestDefaultRunner(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "default command runner")
}

var _ = Describe("Runner Tests", func() {

	It("Invokes RuntimeVisitor and CommandRunner", func() {
		visited := false
		visitor := func(ctx context.Context, runtime *Runtime, command *cobra.Command, args []string) {
			visited = true
		}

		run := false
		runner := func(ctx context.Context, runtime *Runtime, command *cobra.Command, args []string) error {
			run = true
			return nil
		}

		DefaultRunner(visitor, runner)(nil, nil)

		Expect(visited).To(BeTrue())
		Expect(run).To(BeTrue())
	})

	It("Invokes Only CommandRunner if no RuntimeVisitor supplied", func() {
		run := false
		runner := func(ctx context.Context, runtime *Runtime, command *cobra.Command, args []string) error {
			run = true
			return nil
		}

		DefaultRunner(nil, runner)(nil, nil)

		Expect(run).To(BeTrue())
	})

	It("RuntimeWithHyperFleet returns a non-nil visitor that calls WithHyperFleet", func() {
		origExplicitURL := hfExplicitURL
		origLoadConfig := awsLoadConfig
		origGetIdentity := awsGetIdentity
		origNewClient := hfNewClient
		origExitFn := hfExitFn
		defer func() {
			hfExplicitURL = origExplicitURL
			awsLoadConfig = origLoadConfig
			awsGetIdentity = origGetIdentity
			hfNewClient = origNewClient
			hfExitFn = origExitFn
		}()

		hfExplicitURL = func() string { return "https://abc.execute-api.us-east-1.amazonaws.com" }
		awsLoadConfig = func(_ context.Context, _, _ string) (awssdk.Config, error) { return awssdk.Config{}, nil }
		awsGetIdentity = func(_ context.Context, _ awssdk.Config) (*awssts.GetCallerIdentityOutput, error) {
			return &awssts.GetCallerIdentityOutput{
				Account: awssdk.String("123456789012"),
				Arn:     awssdk.String("arn:aws:iam::123456789012:user/test"),
				UserId:  awssdk.String("AIDAEXAMPLE"),
			}, nil
		}
		hfNewClient = func(_ *hfrest.Config) (*hyperfleetclientset.Clientset, error) {
			return &hyperfleetclientset.Clientset{}, nil
		}
		hfExitFn = func(int) {}

		r := &Runtime{Reporter: &reportertest.FakeLogger{}}
		visitor := RuntimeWithHyperFleet()
		Expect(visitor).ToNot(BeNil())
		visitor(nil, r, nil, nil)
		Expect(r.HyperFleetClient).ToNot(BeNil())
	})
})
