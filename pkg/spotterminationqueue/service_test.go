package spotterminationqueue

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	cloudformationtypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeAWSClient struct {
	region           string
	stacks           []*cloudformationtypes.Stack
	createStackErr   error
	getStackErr      error
	getRoleErr       error
	createCallCount  int
	getCallCount     int
	getRoleCallCount int
	lastStackName    string
	lastParams       map[string]string
}

func (f *fakeAWSClient) CreateStackWithParamsTags(_ context.Context, _ string, stackName string,
	stackParams, _ map[string]string) (*string, error) {
	f.createCallCount++
	f.lastStackName = stackName
	f.lastParams = stackParams
	if f.createStackErr != nil {
		return nil, f.createStackErr
	}
	return aws.String(stackName), nil
}

func (f *fakeAWSClient) GetCFStack(_ context.Context, _ string) (*cloudformationtypes.Stack, error) {
	f.getCallCount++
	if f.getStackErr != nil {
		return nil, f.getStackErr
	}
	if len(f.stacks) == 0 {
		return nil, fmt.Errorf("no fake stacks configured")
	}
	stack := f.stacks[0]
	if len(f.stacks) > 1 {
		f.stacks = f.stacks[1:]
	}
	return stack, nil
}

func (f *fakeAWSClient) GetRoleByARN(_ string) (iamtypes.Role, error) {
	f.getRoleCallCount++
	if f.getRoleErr != nil {
		return iamtypes.Role{}, f.getRoleErr
	}
	return iamtypes.Role{}, nil
}

func (f *fakeAWSClient) GetRegion() string {
	return f.region
}

var _ = Describe("stack spec helpers", func() {
	It("builds deterministic resource names from a custom prefix", func() {
		spec := buildStackSpec("demo")
		Expect(spec.NamePrefix).To(Equal("demo"))
		Expect(spec.QueueName).To(Equal("demo-spot-termination-queue"))
		Expect(spec.EventRuleName).To(Equal("demo-spot-termination-events"))
		Expect(spec.StackName).To(Equal("demo-spot-termination-stack"))
	})

	It("falls back to the default prefix when none is provided", func() {
		spec := buildStackSpec("")
		Expect(spec.NamePrefix).To(Equal(defaultNamePrefix))
		Expect(spec.QueueName).To(Equal(defaultNamePrefix + "-queue"))
		Expect(spec.EventRuleName).To(Equal(defaultNamePrefix + "-events"))
		Expect(spec.StackName).To(Equal(defaultNamePrefix + "-stack"))
	})

	It("accepts a name prefix of exactly 40 characters", func() {
		name := "abcdefghij-klmnopqrst-uvwxyz012345678901"
		Expect(len(name)).To(Equal(40))
		Expect(validateNamePrefix(name)).To(Succeed())
	})

	It("rejects a name prefix longer than 40 characters", func() {
		name := "abcdefghij-klmnopqrst-uvwxyz0123456789-xy"
		Expect(len(name)).To(Equal(41))
		Expect(validateNamePrefix(name)).To(MatchError(ContainSubstring("at most 40 characters")))
	})

	It("rejects a name prefix with invalid characters", func() {
		Expect(validateNamePrefix("has_underscore")).To(MatchError(ContainSubstring("letters, numbers, and hyphens")))
		Expect(validateNamePrefix("has space")).To(MatchError(ContainSubstring("letters, numbers, and hyphens")))
	})

	It("accepts an empty name prefix (uses default)", func() {
		Expect(validateNamePrefix("")).To(Succeed())
	})

	It("accepts only auto mode", func() {
		Expect(validateMode("auto")).To(Succeed())
		Expect(validateMode("manual")).To(MatchError(ContainSubstring("only 'auto' is supported")))
	})

	It("creates the queue stack through the injected AWS client", func() {
		queueURL := "https://sqs.us-east-1.amazonaws.com/123456789012/rosa-spot-termination-queue"
		stack := &cloudformationtypes.Stack{
			StackStatus: cloudformationtypes.StackStatusCreateComplete,
			Outputs: []cloudformationtypes.Output{
				{
					OutputKey:   aws.String("QueueUrl"),
					OutputValue: aws.String(queueURL),
				},
			},
		}
		fakeClient := &fakeAWSClient{
			region: "us-east-1",
			stacks: []*cloudformationtypes.Stack{stack, stack},
		}

		result, err := NewService(fakeClient).CreateQueue(context.Background(), CreateInput{
			Name:                      "demo",
			NodePoolManagementRoleArn: "arn:aws:iam::123456789012:role/example-role",
			Mode:                      "auto",
			Region:                    "us-east-1",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(fakeClient.createCallCount).To(Equal(1))
		Expect(fakeClient.lastStackName).To(Equal("demo-spot-termination-stack"))
		Expect(fakeClient.lastParams).To(HaveKeyWithValue("QueueName", "demo-spot-termination-queue"))
		Expect(fakeClient.lastParams).To(HaveKeyWithValue("EventRuleName", "demo-spot-termination-events"))
		Expect(fakeClient.lastParams).To(HaveKeyWithValue(
			"NodePoolManagementRoleArn",
			"arn:aws:iam::123456789012:role/example-role",
		))
		Expect(fakeClient.getRoleCallCount).To(Equal(1))
		Expect(result.QueueURL).To(Equal(queueURL))
		Expect(result.Region).To(Equal("us-east-1"))
	})

	It("fails when the completed stack has no QueueUrl output", func() {
		stack := &cloudformationtypes.Stack{
			StackStatus: cloudformationtypes.StackStatusCreateComplete,
		}
		fakeClient := &fakeAWSClient{
			region: "us-east-1",
			stacks: []*cloudformationtypes.Stack{stack, stack},
		}

		_, err := NewService(fakeClient).CreateQueue(context.Background(), CreateInput{
			Name:                      "demo",
			NodePoolManagementRoleArn: "arn:aws:iam::123456789012:role/example-role",
			Mode:                      "auto",
			Region:                    "us-east-1",
		})
		Expect(err).To(MatchError(ContainSubstring("did not return QueueUrl output")))
	})

	It("fails when the stack ends in a terminal error state", func() {
		fakeClient := &fakeAWSClient{
			region: "us-east-1",
			stacks: []*cloudformationtypes.Stack{{
				StackStatus: cloudformationtypes.StackStatusCreateFailed,
			}},
		}

		_, err := NewService(fakeClient).CreateQueue(context.Background(), CreateInput{
			Name:                      "demo",
			NodePoolManagementRoleArn: "arn:aws:iam::123456789012:role/example-role",
			Mode:                      "auto",
			Region:                    "us-east-1",
		})
		Expect(err).To(MatchError(ContainSubstring("ended in CREATE_FAILED")))
	})

	It("fails when the nodepool management role cannot be validated", func() {
		fakeClient := &fakeAWSClient{
			region:     "us-east-1",
			getRoleErr: fmt.Errorf("role not found"),
		}

		_, err := NewService(fakeClient).CreateQueue(context.Background(), CreateInput{
			Name:                      "demo",
			NodePoolManagementRoleArn: "arn:aws:iam::123456789012:role/example-role",
			Mode:                      "auto",
			Region:                    "us-east-1",
		})
		Expect(err).To(MatchError(ContainSubstring("failed to validate nodepool-management-role-arn")))
		Expect(fakeClient.createCallCount).To(Equal(0))
	})

	It("fails when stack creation fails before polling", func() {
		fakeClient := &fakeAWSClient{
			region:         "us-east-1",
			createStackErr: fmt.Errorf("create failed"),
		}

		_, err := NewService(fakeClient).CreateQueue(context.Background(), CreateInput{
			Name:                      "demo",
			NodePoolManagementRoleArn: "arn:aws:iam::123456789012:role/example-role",
			Mode:                      "auto",
			Region:                    "us-east-1",
		})
		Expect(err).To(MatchError(ContainSubstring("failed to create spot termination stack")))
		Expect(fakeClient.getCallCount).To(Equal(0))
	})

	It("fails when describing the created stack returns an error", func() {
		fakeClient := &fakeAWSClient{
			region:      "us-east-1",
			stacks:      []*cloudformationtypes.Stack{{StackStatus: cloudformationtypes.StackStatusCreateComplete}},
			getStackErr: fmt.Errorf("describe failed"),
		}

		_, err := NewService(fakeClient).CreateQueue(context.Background(), CreateInput{
			Name:                      "demo",
			NodePoolManagementRoleArn: "arn:aws:iam::123456789012:role/example-role",
			Mode:                      "auto",
			Region:                    "us-east-1",
		})
		Expect(err).To(MatchError(ContainSubstring("failed waiting for spot termination stack creation")))
	})
})
