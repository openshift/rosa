package spotterminationqueue

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	rosaaws "github.com/openshift/rosa/pkg/aws"
	opts "github.com/openshift/rosa/pkg/options/spotterminationqueue"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
	queue "github.com/openshift/rosa/pkg/spotterminationqueue"
)

var newSpotTerminationQueueService = func(client rosaaws.Client) queue.Service {
	return queue.NewService(client)
}

// NewCreateSpotTerminationQueueCommand returns the Cobra command for creating Spot termination queue resources.
func NewCreateSpotTerminationQueueCommand() *cobra.Command {
	cmd, options := opts.BuildCreateSpotTerminationQueueCommandWithOptions()
	cmd.Run = rosa.DefaultRunner(rosa.RuntimeWithAWS(), CreateSpotTerminationQueueRunner(options))
	return cmd
}

// CreateSpotTerminationQueueRunner returns a CommandRunner that validates inputs and delegates
// to the spot termination queue service.
func CreateSpotTerminationQueueRunner(userOptions *opts.CreateSpotTerminationQueueUserOptions) rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, _ *cobra.Command, _ []string) error {
		if userOptions.NodePoolManagementRoleArn == "" {
			return fmt.Errorf("you must supply --nodepool-management-role-arn")
		}
		if !rosaaws.RoleArnRE.MatchString(userOptions.NodePoolManagementRoleArn) {
			return fmt.Errorf("expected a valid value for nodepool-management-role-arn matching %s", rosaaws.RoleArnRE)
		}

		result, err := newSpotTerminationQueueService(r.AWSClient).CreateQueue(ctx, queue.CreateInput{
			Name:                      userOptions.Name,
			NodePoolManagementRoleArn: userOptions.NodePoolManagementRoleArn,
			Mode:                      userOptions.Mode,
			Region:                    r.AWSClient.GetRegion(),
		})
		if err != nil {
			return fmt.Errorf("failed to create spot termination queue: %w", err)
		}

		if output.HasFlag() {
			if err := output.Print(result); err != nil {
				return fmt.Errorf("failed to print spot termination queue result: %w", err)
			}
			return nil
		}

		r.Reporter.Infof("Created spot termination queue resources in region '%s'", result.Region)
		r.Reporter.Infof("CloudFormation stack: %s", result.StackName)
		r.Reporter.Infof("Queue name: %s", result.QueueName)
		r.Reporter.Infof("EventBridge rule: %s", result.EventRuleName)
		r.Reporter.Infof("Queue URL: %s", result.QueueURL)
		r.Reporter.Infof(
			"Use this queue URL with 'rosa create cluster --spot-termination-queue-url %s' "+
				"or 'rosa edit cluster --spot-termination-queue-url %s --cluster <cluster>'",
			result.QueueURL,
			result.QueueURL,
		)
		return nil
	}
}
