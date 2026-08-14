package spotterminationqueue

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/output"
)

// CreateSpotTerminationQueueUserOptions holds user-supplied flag values for the
// spot-termination-queue create command.
type CreateSpotTerminationQueueUserOptions struct {
	Name                      string
	NodePoolManagementRoleArn string
	Mode                      string
}

const (
	use     = "spot-termination-queue"
	short   = "Create Spot termination queue resources"
	long    = "Create the SQS queue, queue policy, and EventBridge rule needed for ROSA HCP Spot interruption handling."
	example = `  # Create the Spot termination queue resources using a custom resource-name prefix
  rosa create spot-termination-queue --name my-cluster-prefix \
    --nodepool-management-role-arn arn:aws:iam::123456789012:role/example-role

  # Print the created queue metadata as JSON
  rosa create spot-termination-queue --name my-cluster-prefix \
    --nodepool-management-role-arn arn:aws:iam::123456789012:role/example-role -o json`
)

// NewCreateSpotTerminationQueueUserOptions returns options pre-populated with default values.
func NewCreateSpotTerminationQueueUserOptions() *CreateSpotTerminationQueueUserOptions {
	return &CreateSpotTerminationQueueUserOptions{
		Mode: "auto",
	}
}

// BuildCreateSpotTerminationQueueCommandWithOptions returns a Cobra command wired to
// the returned user options struct for flag binding.
func BuildCreateSpotTerminationQueueCommandWithOptions() (*cobra.Command, *CreateSpotTerminationQueueUserOptions) {
	options := NewCreateSpotTerminationQueueUserOptions()
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Example: example,
		Args:    cobra.NoArgs,
	}

	flags := cmd.Flags()
	flags.StringVar(
		&options.Name,
		"name",
		"",
		"Optional resource name prefix used to derive the queue, stack, and event rule names. "+
			"When omitted, ROSA uses a default spot-termination prefix.",
	)
	flags.StringVar(
		&options.NodePoolManagementRoleArn,
		"nodepool-management-role-arn",
		"",
		"ARN of the Hosted Control Plane NodePoolManagement role that must be allowed to receive and delete SQS messages.",
	)
	flags.StringVar(
		&options.Mode,
		"mode",
		"auto",
		"Queue setup mode. Currently only 'auto' is supported.",
	)
	output.AddFlag(cmd)

	return cmd, options
}
