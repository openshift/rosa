package aws

import (
	"context"
	"fmt"

	gomock "go.uber.org/mock/gomock"

	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cloudformationtypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/aws/mocks"
)

var _ = Describe("CloudFormation", func() {
	var (
		client    Client
		mockCtrl  *gomock.Controller
		mockCfAPI *mocks.MockCloudFormationApiClient
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCfAPI = mocks.NewMockCloudFormationApiClient(mockCtrl)
		client = New(
			awsSdk.Config{},
			NewLoggerWrapper(logrus.New(), nil),
			mocks.NewMockIamApiClient(mockCtrl),
			mocks.NewMockEc2ApiClient(mockCtrl),
			mocks.NewMockOrganizationsApiClient(mockCtrl),
			mocks.NewMockS3ApiClient(mockCtrl),
			mocks.NewMockSecretsManagerApiClient(mockCtrl),
			mocks.NewMockStsApiClient(mockCtrl),
			mockCfAPI,
			mocks.NewMockServiceQuotasApiClient(mockCtrl),
			mocks.NewMockServiceQuotasApiClient(mockCtrl),
			&AccessKey{},
			false,
		)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("CheckStackReadyOrNotExisting", func() {
		stackName := "test-stack"

		It("Returns ready when stack is CREATE_COMPLETE", func() {
			mockCfAPI.EXPECT().ListStacks(gomock.Any(), gomock.Any()).Return(
				&cloudformation.ListStacksOutput{
					StackSummaries: []cloudformationtypes.StackSummary{
						{
							StackName:   awsSdk.String(stackName),
							StackStatus: cloudformationtypes.StackStatusCreateComplete,
						},
					},
				}, nil)

			ready, status, err := client.CheckStackReadyOrNotExisting(stackName)
			Expect(err).NotTo(HaveOccurred())
			Expect(ready).To(BeTrue())
			Expect(*status).To(Equal("CREATE_COMPLETE"))
		})

		It("Returns ready when stack is UPDATE_COMPLETE", func() {
			mockCfAPI.EXPECT().ListStacks(gomock.Any(), gomock.Any()).Return(
				&cloudformation.ListStacksOutput{
					StackSummaries: []cloudformationtypes.StackSummary{
						{
							StackName:   awsSdk.String(stackName),
							StackStatus: cloudformationtypes.StackStatusUpdateComplete,
						},
					},
				}, nil)

			ready, status, err := client.CheckStackReadyOrNotExisting(stackName)
			Expect(err).NotTo(HaveOccurred())
			Expect(ready).To(BeTrue())
			Expect(*status).To(Equal("UPDATE_COMPLETE"))
		})

		It("Skips DELETE_COMPLETE stacks (treated as non-existing)", func() {
			mockCfAPI.EXPECT().ListStacks(gomock.Any(), gomock.Any()).Return(
				&cloudformation.ListStacksOutput{
					StackSummaries: []cloudformationtypes.StackSummary{
						{
							StackName:   awsSdk.String(stackName),
							StackStatus: cloudformationtypes.StackStatusDeleteComplete,
						},
					},
				}, nil)

			ready, status, err := client.CheckStackReadyOrNotExisting(stackName)
			Expect(err).NotTo(HaveOccurred())
			Expect(ready).To(BeFalse())
			Expect(status).To(BeNil())
		})

		It("Returns error with suggestion for ROLLBACK_COMPLETE stack", func() {
			mockCfAPI.EXPECT().ListStacks(gomock.Any(), gomock.Any()).Return(
				&cloudformation.ListStacksOutput{
					StackSummaries: []cloudformationtypes.StackSummary{
						{
							StackName:   awsSdk.String(stackName),
							StackStatus: cloudformationtypes.StackStatusRollbackComplete,
						},
					},
				}, nil)

			ready, status, err := client.CheckStackReadyOrNotExisting(stackName)
			Expect(err).To(HaveOccurred())
			Expect(ready).To(BeFalse())
			Expect(*status).To(Equal("ROLLBACK_COMPLETE"))
			Expect(err.Error()).To(ContainSubstring(
				"exists with status ROLLBACK_COMPLETE. Expected status is CREATE_COMPLETE"))
			Expect(err.Error()).To(ContainSubstring("rosa init --delete-stack"))
		})

		It("Returns not found when no stacks match", func() {
			mockCfAPI.EXPECT().ListStacks(gomock.Any(), gomock.Any()).Return(
				&cloudformation.ListStacksOutput{
					StackSummaries: []cloudformationtypes.StackSummary{
						{
							StackName:   awsSdk.String("other-stack"),
							StackStatus: cloudformationtypes.StackStatusCreateComplete,
						},
					},
				}, nil)

			ready, status, err := client.CheckStackReadyOrNotExisting(stackName)
			Expect(err).NotTo(HaveOccurred())
			Expect(ready).To(BeFalse())
			Expect(status).To(BeNil())
		})

		It("Returns not found when stack list is empty", func() {
			mockCfAPI.EXPECT().ListStacks(gomock.Any(), gomock.Any()).Return(
				&cloudformation.ListStacksOutput{
					StackSummaries: []cloudformationtypes.StackSummary{},
				}, nil)

			ready, status, err := client.CheckStackReadyOrNotExisting(stackName)
			Expect(err).NotTo(HaveOccurred())
			Expect(ready).To(BeFalse())
			Expect(status).To(BeNil())
		})

		It("Propagates ListStacks API error", func() {
			mockCfAPI.EXPECT().ListStacks(gomock.Any(), gomock.Any()).Return(
				nil, fmt.Errorf("access denied"))

			ready, status, err := client.CheckStackReadyOrNotExisting(stackName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("access denied"))
			Expect(ready).To(BeFalse())
			Expect(status).To(BeNil())
		})
	})

	Context("GetCFStack", func() {
		stackName := "test-stack"
		ctx := context.Background()

		It("Returns the first stack on success", func() {
			expected := cloudformationtypes.Stack{
				StackName:   awsSdk.String(stackName),
				StackStatus: cloudformationtypes.StackStatusCreateComplete,
			}
			mockCfAPI.EXPECT().DescribeStacks(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, input *cloudformation.DescribeStacksInput,
					_ ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
					Expect(*input.StackName).To(Equal(stackName))
					return &cloudformation.DescribeStacksOutput{
						Stacks: []cloudformationtypes.Stack{expected},
					}, nil
				})

			stack, err := client.GetCFStack(ctx, stackName)
			Expect(err).NotTo(HaveOccurred())
			Expect(*stack.StackName).To(Equal(stackName))
			Expect(stack.StackStatus).To(Equal(cloudformationtypes.StackStatusCreateComplete))
		})

		It("Returns error when no stacks found", func() {
			mockCfAPI.EXPECT().DescribeStacks(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ *cloudformation.DescribeStacksInput,
					_ ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
					return &cloudformation.DescribeStacksOutput{
						Stacks: []cloudformationtypes.Stack{},
					}, nil
				})

			stack, err := client.GetCFStack(ctx, stackName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No CF stacks with name"))
			Expect(err.Error()).To(ContainSubstring(stackName))
			Expect(stack).To(BeNil())
		})

		It("Propagates DescribeStacks API error", func() {
			mockCfAPI.EXPECT().DescribeStacks(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ *cloudformation.DescribeStacksInput,
					_ ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
					return nil, fmt.Errorf("stack not found")
				})

			stack, err := client.GetCFStack(ctx, stackName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stack not found"))
			Expect(stack).To(BeNil())
		})
	})

	Context("DescribeCFStackResources", func() {
		stackName := "test-stack"
		ctx := context.Background()

		It("Returns stack resources on success", func() {
			resources := []cloudformationtypes.StackResource{
				{
					LogicalResourceId: awsSdk.String("AdminUser"),
					ResourceType:      awsSdk.String("AWS::IAM::User"),
					ResourceStatus:    cloudformationtypes.ResourceStatusCreateComplete,
				},
				{
					LogicalResourceId: awsSdk.String("AdminPolicy"),
					ResourceType:      awsSdk.String("AWS::IAM::Policy"),
					ResourceStatus:    cloudformationtypes.ResourceStatusCreateComplete,
				},
			}
			mockCfAPI.EXPECT().DescribeStackResources(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, input *cloudformation.DescribeStackResourcesInput,
					_ ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
					Expect(*input.StackName).To(Equal(stackName))
					return &cloudformation.DescribeStackResourcesOutput{
						StackResources: resources,
					}, nil
				})

			result, err := client.DescribeCFStackResources(ctx, stackName)
			Expect(err).NotTo(HaveOccurred())
			Expect(*result).To(HaveLen(2))
			Expect(*(*result)[0].LogicalResourceId).To(Equal("AdminUser"))
		})

		It("Propagates DescribeStackResources API error", func() {
			mockCfAPI.EXPECT().DescribeStackResources(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ *cloudformation.DescribeStackResourcesInput,
					_ ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
					return nil, fmt.Errorf("access denied")
				})

			result, err := client.DescribeCFStackResources(ctx, stackName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("access denied"))
			Expect(result).To(BeNil())
		})
	})

	Context("DeleteCFStack", func() {
		stackName := "test-stack"
		ctx := context.Background()

		It("Succeeds when DeleteStack returns no error", func() {
			mockCfAPI.EXPECT().DeleteStack(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, input *cloudformation.DeleteStackInput,
					_ ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error) {
					Expect(*input.StackName).To(Equal(stackName))
					return &cloudformation.DeleteStackOutput{}, nil
				})

			err := client.DeleteCFStack(ctx, stackName)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Propagates DeleteStack API error", func() {
			mockCfAPI.EXPECT().DeleteStack(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ *cloudformation.DeleteStackInput,
					_ ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error) {
					return nil, fmt.Errorf("stack in use")
				})

			err := client.DeleteCFStack(ctx, stackName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stack in use"))
		})
	})

	Context("DeleteOsdCcsAdminUser", func() {
		stackName := "test-stack"

		It("Returns nil on TokenAlreadyExistsException", func() {
			mockCfAPI.EXPECT().DeleteStack(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ *cloudformation.DeleteStackInput,
					_ ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error) {
					return nil, &cloudformationtypes.TokenAlreadyExistsException{
						Message: awsSdk.String("token exists"),
					}
				})

			err := client.DeleteOsdCcsAdminUser(stackName)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Propagates non-TokenAlreadyExistsException errors", func() {
			mockCfAPI.EXPECT().DeleteStack(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ *cloudformation.DeleteStackInput,
					_ ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error) {
					return nil, fmt.Errorf("internal server error")
				})

			err := client.DeleteOsdCcsAdminUser(stackName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("internal server error"))
		})
	})

	Context("UpdateStack", func() {
		stackName := "test-stack"
		templateBody := `{"AWSTemplateFormatVersion":"2010-09-09"}`

		It("Returns nil when 'No updates are to be performed'", func() {
			mockCfAPI.EXPECT().UpdateStack(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, input *cloudformation.UpdateStackInput,
					_ ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error) {
					Expect(*input.StackName).To(Equal(stackName))
					Expect(*input.TemplateBody).To(Equal(templateBody))
					return nil, &smithy.GenericAPIError{
						Code:    "ValidationError",
						Message: "No updates are to be performed",
					}
				})

			awsC := client.(*awsClient)
			err := awsC.UpdateStack(templateBody, stackName)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Returns error for non-no-op ValidationError", func() {
			mockCfAPI.EXPECT().UpdateStack(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ *cloudformation.UpdateStackInput,
					_ ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error) {
					return nil, &smithy.GenericAPIError{
						Code:    "ValidationError",
						Message: "Template format error",
					}
				})

			awsC := client.(*awsClient)
			err := awsC.UpdateStack(templateBody, stackName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Template format error"))
		})

		It("Returns error for non-ValidationError API error", func() {
			mockCfAPI.EXPECT().UpdateStack(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ *cloudformation.UpdateStackInput,
					_ ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error) {
					return nil, fmt.Errorf("throttling exception")
				})

			awsC := client.(*awsClient)
			err := awsC.UpdateStack(templateBody, stackName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("throttling exception"))
		})
	})

	Context("buildCreateStackInput", func() {
		It("Sets correct capabilities, name, and template", func() {
			body := `{"Resources":{}}`
			name := "my-stack"
			params := []cloudformationtypes.Parameter{
				{ParameterKey: awsSdk.String("Env"), ParameterValue: awsSdk.String("prod")},
			}
			tags := []cloudformationtypes.Tag{
				{Key: awsSdk.String("team"), Value: awsSdk.String("platform")},
			}

			input := buildCreateStackInput(body, name, params, tags)

			Expect(*input.StackName).To(Equal(name))
			Expect(*input.TemplateBody).To(Equal(body))
			Expect(input.Capabilities).To(HaveLen(3))
			Expect(input.Capabilities).To(ContainElements(
				cloudformationtypes.CapabilityCapabilityIam,
				cloudformationtypes.CapabilityCapabilityNamedIam,
				cloudformationtypes.CapabilityCapabilityAutoExpand,
			))
			Expect(input.Parameters).To(HaveLen(1))
			Expect(*input.Parameters[0].ParameterKey).To(Equal("Env"))
			Expect(*input.Parameters[0].ParameterValue).To(Equal("prod"))
			Expect(input.Tags).To(HaveLen(1))
			Expect(*input.Tags[0].Key).To(Equal("team"))
			Expect(*input.Tags[0].Value).To(Equal("platform"))
		})

		It("Handles empty params and tags", func() {
			input := buildCreateStackInput("body", "stack", nil, nil)

			Expect(*input.StackName).To(Equal("stack"))
			Expect(*input.TemplateBody).To(Equal("body"))
			Expect(input.Capabilities).To(HaveLen(3))
			Expect(input.Parameters).To(BeNil())
			Expect(input.Tags).To(BeNil())
		})
	})

	Context("buildUpdateStackInput", func() {
		It("Sets correct capabilities, name, and template", func() {
			body := `{"Resources":{}}`
			name := "my-stack"

			input := buildUpdateStackInput(body, name)

			Expect(*input.StackName).To(Equal(name))
			Expect(*input.TemplateBody).To(Equal(body))
			Expect(input.Capabilities).To(HaveLen(3))
			Expect(input.Capabilities).To(ContainElements(
				cloudformationtypes.CapabilityCapabilityIam,
				cloudformationtypes.CapabilityCapabilityNamedIam,
				cloudformationtypes.CapabilityCapabilityAutoExpand,
			))
		})
	})
})
