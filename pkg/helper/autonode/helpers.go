package autonode

import (
	"fmt"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/fedramp"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	// AutoNodeFlagName is the flag name for enabling/configuring AutoNode
	AutoNodeFlagName = "autonode"
	// AutoNodeIAMRoleArnFlagName is the flag name for the AutoNode IAM role ARN
	AutoNodeIAMRoleArnFlagName = "autonode-iam-role-arn"
)

// AutoNodeConfig holds AutoNode configuration parameters and results
type AutoNodeConfig struct {
	// Input parameters
	AutoNodeFlag string // Value from --autonode flag
	RoleARNFlag  string // Value from --autonode-iam-role-arn flag

	// Output values
	AutoNodeMode    string // AutoNode mode to set (e.g., "enabled")
	AutoNodeRoleARN string // IAM role ARN to use
}

// ValidateAutoNodeValue validates the autonode flag value
func ValidateAutoNodeValue(value string) error {
	if value != "enabled" {
		return fmt.Errorf("Invalid value for --autonode. Currently only 'enabled' is supported")
	}
	return nil
}

// ValidateRoleARN validates the IAM role ARN format
func ValidateRoleARN(roleArn string) error {
	if roleArn == "" {
		return fmt.Errorf("IAM role ARN cannot be empty")
	}
	if !aws.RoleArnRE.MatchString(roleArn) {
		return fmt.Errorf("Invalid IAM role ARN format: '%s'. Expected format: arn:aws:iam::<account-id>:role/<role-name>", roleArn)
	}
	return nil
}

// ValidateAutoNodeConfiguration validates the overall AutoNode configuration state
func ValidateAutoNodeConfiguration(autoNodeChanged, roleArnChanged, currentEnabled bool, roleArnValue string) error {
	// Validate enabling AutoNode when already enabled
	if autoNodeChanged && currentEnabled {
		return fmt.Errorf("AutoNode is already enabled for this cluster")
	}

	// Validate IAM role ARN is provided when enabling AutoNode
	if autoNodeChanged && (!roleArnChanged || roleArnValue == "") {
		return fmt.Errorf("IAM role ARN is required when enabling AutoNode")
	}

	// Validate can't update IAM role when AutoNode is not enabled
	if roleArnChanged && !autoNodeChanged && !currentEnabled {
		return fmt.Errorf("Cannot update IAM role ARN when AutoNode is not enabled. " +
			"Enable AutoNode first with --autonode=enabled")
	}

	return nil
}

// DetermineAutoNodeMode determines the AutoNode mode to set
func DetermineAutoNodeMode(autoNodeChanged bool, flagValue string) string {
	// Only set mode if explicitly provided by user
	if autoNodeChanged {
		return flagValue
	}
	// If user is only updating IAM role, don't set mode (backend maintains current state)
	return ""
}

// SetAutoNode validates and sets AutoNode configuration for a cluster
func SetAutoNode(r *rosa.Runtime, cmd *cobra.Command, cluster *cmv1.Cluster, flagValue string, roleArnValue string) (*AutoNodeConfig, error) {
	config := &AutoNodeConfig{
		AutoNodeFlag: flagValue,
		RoleARNFlag:  roleArnValue,
	}

	// Check if flags are changed
	autoNodeChanged := cmd.Flags().Changed(AutoNodeFlagName)
	roleArnChanged := cmd.Flags().Changed(AutoNodeIAMRoleArnFlagName)

	if !autoNodeChanged && !roleArnChanged {
		return nil, nil
	}

	// Validate HCP cluster
	if !aws.IsHostedCP(cluster) {
		return nil, fmt.Errorf("AutoNode is only supported for Hosted Control Plane clusters")
	}

	currentAutoNodeMode, currentAutoNodeExists := ocm.GetAutoNodeMode(cluster)
	currentAutoNodeEnabled := currentAutoNodeExists && currentAutoNodeMode == "enabled"

	// Validate autonode value once if provided
	if autoNodeChanged {
		if err := ValidateAutoNodeValue(config.AutoNodeFlag); err != nil {
			return nil, err
		}
	}

	// Validate overall configuration
	if err := ValidateAutoNodeConfiguration(autoNodeChanged, roleArnChanged, currentAutoNodeEnabled, config.RoleARNFlag); err != nil {
		return nil, err
	}

	// Validate IAM role ARN format if provided
	if roleArnChanged {
		if err := ValidateRoleARN(config.RoleARNFlag); err != nil {
			return nil, err
		}
		config.AutoNodeRoleARN = config.RoleARNFlag
	}

	// Set the AutoNode mode if explicitly provided
	config.AutoNodeMode = DetermineAutoNodeMode(autoNodeChanged, config.AutoNodeFlag)

	// No confirmation prompts needed - AutoNode enablement doesn't cause disruption

	return config, nil
}

// InteractivePrompt handles interactive mode for AutoNode configuration
func InteractivePrompt(r *rosa.Runtime, cmd *cobra.Command, cluster *cmv1.Cluster) (*AutoNodeConfig, error) {
	// AutoNode is not supported in govcloud - skip prompting
	if fedramp.Enabled() {
		return nil, nil
	}

	config := &AutoNodeConfig{}

	autoNodeMode, autoNodeExists := ocm.GetAutoNodeMode(cluster)
	autoNodeEnabled := autoNodeExists && autoNodeMode == "enabled"
	currentRoleArn, _ := ocm.GetAutoNodeRoleArn(cluster)

	// Build the appropriate question based on current state
	var question string
	if autoNodeEnabled {
		question = fmt.Sprintf("Update AutoNode IAM role ARN (current: %s)", currentRoleArn)
	} else {
		question = "Enable AutoNode"
	}

	// First prompt: enable/update decision
	proceed, err := interactive.GetBool(interactive.Input{
		Question: question,
		Default:  false,
		Required: false,
	})
	if err != nil {
		return nil, err
	}

	if !proceed {
		return nil, nil
	}

	// Only set mode when enabling, not when updating role
	if !autoNodeEnabled {
		config.AutoNodeMode = "enabled"
	}

	// Build role ARN prompt based on state
	roleQuestion := "AutoNode IAM role ARN"
	if autoNodeEnabled {
		roleQuestion = "New AutoNode IAM role ARN"
	}

	// Second prompt: IAM role ARN
	roleArnValue, err := interactive.GetString(interactive.Input{
		Question: roleQuestion,
		Help:     cmd.Flags().Lookup(AutoNodeIAMRoleArnFlagName).Usage,
		Default:  currentRoleArn, // Empty string if not enabled
		Required: true,
		Validators: []interactive.Validator{
			interactive.RegExp(aws.RoleArnRE.String()),
		},
	})
	if err != nil {
		return nil, err
	}
	config.AutoNodeRoleARN = strings.TrimSpace(roleArnValue)

	return config, nil
}
