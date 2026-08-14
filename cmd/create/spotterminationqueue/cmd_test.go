package spotterminationqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mock "github.com/openshift/rosa/pkg/aws"
	opts "github.com/openshift/rosa/pkg/options/spotterminationqueue"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
	queue "github.com/openshift/rosa/pkg/spotterminationqueue"
)

func TestSpotTerminationQueueCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Spot termination queue command suite")
}

type fakeQueueService struct {
	input       queue.CreateInput
	result      *queue.Result
	err         error
	invocations int
}

func (f *fakeQueueService) CreateQueue(_ context.Context, input queue.CreateInput) (*queue.Result, error) {
	f.invocations++
	f.input = input
	return f.result, f.err
}

var _ = Describe("CreateSpotTerminationQueueRunner", func() {
	var (
		ctrl        *gomock.Controller
		mockClient  *mock.MockClient
		fakeService *fakeQueueService
		runtime     *rosa.Runtime
		oldFactory  func(mock.Client) queue.Service
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClient = mock.NewMockClient(ctrl)
		mockClient.EXPECT().GetRegion().Return("us-east-1").AnyTimes()

		fakeService = &fakeQueueService{
			result: &queue.Result{
				QueueURL:      "https://sqs.us-east-1.amazonaws.com/123456789012/demo",
				QueueName:     "demo-spot-termination-queue",
				EventRuleName: "demo-spot-termination-events",
				StackName:     "demo-spot-termination-stack",
				Region:        "us-east-1",
			},
		}

		oldFactory = newSpotTerminationQueueService
		newSpotTerminationQueueService = func(client mock.Client) queue.Service {
			return fakeService
		}

		runtime = rosa.NewRuntime()
		runtime.AWSClient = mockClient
	})

	AfterEach(func() {
		newSpotTerminationQueueService = oldFactory
		ctrl.Finish()
	})

	Context("validation", func() {
		It("rejects a missing nodepool-management-role-arn", func() {
			options := &opts.CreateSpotTerminationQueueUserOptions{
				Name: "demo",
				Mode: "auto",
			}

			runner := CreateSpotTerminationQueueRunner(options)
			err := runner(context.Background(), runtime, NewCreateSpotTerminationQueueCommand(), []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("--nodepool-management-role-arn"))
			Expect(fakeService.invocations).To(Equal(0))
		})

		It("rejects an invalid nodepool-management-role-arn", func() {
			options := &opts.CreateSpotTerminationQueueUserOptions{
				Name:                      "demo",
				NodePoolManagementRoleArn: "not-a-valid-arn",
				Mode:                      "auto",
			}

			runner := CreateSpotTerminationQueueRunner(options)
			err := runner(context.Background(), runtime, NewCreateSpotTerminationQueueCommand(), []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("valid value for nodepool-management-role-arn"))
			Expect(fakeService.invocations).To(Equal(0))
		})
	})

	Context("successful execution", func() {
		It("passes options through to the service", func() {
			options := &opts.CreateSpotTerminationQueueUserOptions{
				Name:                      "demo",
				NodePoolManagementRoleArn: "arn:aws:iam::123456789012:role/example-role",
				Mode:                      "auto",
			}

			runner := CreateSpotTerminationQueueRunner(options)
			err := runner(context.Background(), runtime, NewCreateSpotTerminationQueueCommand(), []string{})
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeService.invocations).To(Equal(1))
			Expect(fakeService.input.Name).To(Equal("demo"))
			Expect(fakeService.input.NodePoolManagementRoleArn).To(Equal("arn:aws:iam::123456789012:role/example-role"))
			Expect(fakeService.input.Mode).To(Equal("auto"))
			Expect(fakeService.input.Region).To(Equal("us-east-1"))
		})

		It("supports JSON output mode", func() {
			options := &opts.CreateSpotTerminationQueueUserOptions{
				Name:                      "demo",
				NodePoolManagementRoleArn: "arn:aws:iam::123456789012:role/example-role",
				Mode:                      "auto",
			}

			cmd := NewCreateSpotTerminationQueueCommand()
			output.SetOutput("json")
			defer output.SetOutput("")

			stdoutState := os.Stdout
			stdoutReader, stdoutWriter, err := os.Pipe()
			Expect(err).ToNot(HaveOccurred())
			os.Stdout = stdoutWriter
			defer func() {
				os.Stdout = stdoutState
			}()

			runner := CreateSpotTerminationQueueRunner(options)
			err = runner(context.Background(), runtime, cmd, []string{})
			Expect(err).ToNot(HaveOccurred())

			Expect(stdoutWriter.Close()).To(Succeed())
			stdout, err := io.ReadAll(stdoutReader)
			Expect(err).ToNot(HaveOccurred())

			var result queue.Result
			Expect(json.Unmarshal(stdout, &result)).To(Succeed())
			Expect(result.QueueURL).To(Equal(fakeService.result.QueueURL))
			Expect(result.QueueName).To(Equal(fakeService.result.QueueName))
			Expect(result.EventRuleName).To(Equal(fakeService.result.EventRuleName))
			Expect(result.StackName).To(Equal(fakeService.result.StackName))
			Expect(result.Region).To(Equal(fakeService.result.Region))
		})
	})

	Context("error handling", func() {
		It("wraps and returns service errors", func() {
			fakeService.result = nil
			fakeService.err = fmt.Errorf("stack creation failed")

			options := &opts.CreateSpotTerminationQueueUserOptions{
				Name:                      "demo",
				NodePoolManagementRoleArn: "arn:aws:iam::123456789012:role/example-role",
				Mode:                      "auto",
			}

			runner := CreateSpotTerminationQueueRunner(options)
			err := runner(context.Background(), runtime, NewCreateSpotTerminationQueueCommand(), []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to create spot termination queue: stack creation failed"))
		})
	})
})
