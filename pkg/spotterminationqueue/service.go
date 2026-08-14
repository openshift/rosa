package spotterminationqueue

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	cloudformationtypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
)

const defaultNamePrefix = "rosa-spot-termination"

var namePrefixRE = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

// CreateInput holds the parameters for creating Spot termination queue resources.
type CreateInput struct {
	Name                      string
	NodePoolManagementRoleArn string
	Mode                      string
	Region                    string
}

// Result describes the resources created by a Spot termination queue operation.
type Result struct {
	QueueURL      string `json:"queue_url"`
	QueueName     string `json:"queue_name"`
	EventRuleName string `json:"event_rule_name"`
	StackName     string `json:"stack_name"`
	Region        string `json:"region"`
}

type stackSpec struct {
	NamePrefix    string
	QueueName     string
	EventRuleName string
	StackName     string
}

type awsClient interface {
	CreateStackWithParamsTags(ctx context.Context, cfTemplateBody, stackName string,
		stackParams, stackTags map[string]string) (*string, error)
	GetCFStack(ctx context.Context, stackName string) (*cloudformationtypes.Stack, error)
	GetRoleByARN(roleARN string) (iamtypes.Role, error)
	GetRegion() string
}

// Service manages the lifecycle of Spot termination queue AWS resources.
type Service interface {
	CreateQueue(ctx context.Context, input CreateInput) (*Result, error)
}

type service struct {
	awsClient awsClient
}

// NewService returns a Service backed by the given AWS client.
func NewService(awsClient awsClient) Service {
	return &service{
		awsClient: awsClient,
	}
}

func validateMode(mode string) error {
	if mode == "" || mode == "auto" {
		return nil
	}
	return fmt.Errorf("only 'auto' is supported for --mode")
}

func validateNamePrefix(name string) error {
	if name == "" {
		return nil
	}
	const maxNamePrefixLength = 40
	if len(name) > maxNamePrefixLength {
		return fmt.Errorf("name must be at most %d characters (EventBridge rule name limit)", maxNamePrefixLength)
	}
	if !namePrefixRE.MatchString(name) {
		return fmt.Errorf("name must contain only letters, numbers, and hyphens")
	}
	return nil
}

func buildStackSpec(name string) stackSpec {
	if name == "" {
		return stackSpec{
			NamePrefix:    defaultNamePrefix,
			QueueName:     defaultNamePrefix + "-queue",
			EventRuleName: defaultNamePrefix + "-events",
			StackName:     defaultNamePrefix + "-stack",
		}
	}

	baseName := name + "-spot-termination"
	return stackSpec{
		NamePrefix:    name,
		QueueName:     baseName + "-queue",
		EventRuleName: baseName + "-events",
		StackName:     baseName + "-stack",
	}
}

func (s *service) CreateQueue(ctx context.Context, input CreateInput) (*Result, error) {
	if err := validateMode(input.Mode); err != nil {
		return nil, err
	}
	if err := validateNamePrefix(input.Name); err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.NodePoolManagementRoleArn) == "" {
		return nil, fmt.Errorf("nodepool-management-role-arn is required")
	}
	if s.awsClient == nil {
		return nil, fmt.Errorf("aws client is required")
	}
	if _, err := s.awsClient.GetRoleByARN(input.NodePoolManagementRoleArn); err != nil {
		return nil, fmt.Errorf("failed to validate nodepool-management-role-arn: %w", err)
	}

	spec := buildStackSpec(input.Name)

	stackParams := map[string]string{
		"QueueName":                 spec.QueueName,
		"EventRuleName":             spec.EventRuleName,
		"NodePoolManagementRoleArn": input.NodePoolManagementRoleArn,
	}

	_, err := s.awsClient.CreateStackWithParamsTags(
		ctx,
		cloudFormationTemplate,
		spec.StackName,
		stackParams,
		map[string]string{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create spot termination stack: %w", err)
	}

	err = waitForStackCreateComplete(ctx, s.awsClient, spec.StackName)
	if err != nil {
		return nil, fmt.Errorf("failed waiting for spot termination stack creation: %w", err)
	}

	describe, err := s.awsClient.GetCFStack(ctx, spec.StackName)
	if err != nil {
		return nil, fmt.Errorf("failed to describe spot termination stack: %w", err)
	}

	queueURL := ""
	for _, output := range describe.Outputs {
		if output.OutputKey != nil && *output.OutputKey == "QueueUrl" && output.OutputValue != nil {
			queueURL = *output.OutputValue
			break
		}
	}
	if queueURL == "" {
		return nil, fmt.Errorf("spot termination stack did not return QueueUrl output")
	}

	return &Result{
		QueueURL:      queueURL,
		QueueName:     spec.QueueName,
		EventRuleName: spec.EventRuleName,
		StackName:     spec.StackName,
		Region:        s.awsClient.GetRegion(),
	}, nil
}

func waitForStackCreateComplete(ctx context.Context, awsClient awsClient, stackName string) error {
	deadline := time.Now().Add(10 * time.Minute)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for CloudFormation stack %s", stackName)
		}

		stack, err := awsClient.GetCFStack(ctx, stackName)
		if err != nil {
			return err
		}

		switch stack.StackStatus {
		case cloudformationtypes.StackStatusCreateComplete, cloudformationtypes.StackStatusUpdateComplete:
			return nil
		case cloudformationtypes.StackStatusCreateFailed, cloudformationtypes.StackStatusRollbackComplete,
			cloudformationtypes.StackStatusRollbackFailed, cloudformationtypes.StackStatusDeleteComplete,
			cloudformationtypes.StackStatusDeleteFailed:
			return fmt.Errorf("stack %s ended in %s", stackName, stack.StackStatus)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

const cloudFormationTemplate = `AWSTemplateFormatVersion: '2010-09-09'
Description: ROSA HCP Spot termination queue resources
Parameters:
  QueueName:
    Type: String
  EventRuleName:
    Type: String
  NodePoolManagementRoleArn:
    Type: String
Resources:
  SpotTerminationQueue:
    Type: AWS::SQS::Queue
    Properties:
      QueueName: !Ref QueueName
      Tags:
        - Key: red-hat
          Value: "true"
  SpotTerminationQueuePolicy:
    Type: AWS::SQS::QueuePolicy
    Properties:
      Queues:
        - !Ref SpotTerminationQueue
      PolicyDocument:
        Version: "2012-10-17"
        Statement:
          - Sid: AllowNodePoolManagementRole
            Effect: Allow
            Principal:
              AWS: !Ref NodePoolManagementRoleArn
            Action:
              - sqs:DeleteMessage
              - sqs:ReceiveMessage
            Resource: !GetAtt SpotTerminationQueue.Arn
          - Sid: AllowEventBridgeToSendMessages
            Effect: Allow
            Principal:
              Service: events.amazonaws.com
            Action:
              - sqs:SendMessage
            Resource: !GetAtt SpotTerminationQueue.Arn
            Condition:
              ArnEquals:
                aws:SourceArn: !GetAtt SpotTerminationEventRule.Arn
  SpotTerminationEventRule:
    Type: AWS::Events::Rule
    Properties:
      Name: !Ref EventRuleName
      EventPattern:
        source:
          - aws.ec2
        detail-type:
          - EC2 Spot Instance Interruption Warning
          - EC2 Instance Rebalance Recommendation
      Targets:
        - Arn: !GetAtt SpotTerminationQueue.Arn
          Id: spot-termination-queue
Outputs:
  QueueUrl:
    Value: !Ref SpotTerminationQueue
  QueueArn:
    Value: !GetAtt SpotTerminationQueue.Arn
`
