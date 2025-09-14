package autonode

import (
	"fmt"
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

// SetAutoNode validates and sets AutoNode configuration for a cluster
func SetAutoNode(r *rosa.Runtime, cmd *cobra.Command, cluster *cmv1.Cluster, autonode string, roleArn string) (
	autonodeMode string, autonodeRole string, err error) {

	// Check if flags are changed
	autonodeChanged := cmd.Flags().Changed("autonode")
	roleArnChanged := cmd.Flags().Changed("autonode-iam-role-arn")

	if !autonodeChanged && !roleArnChanged {
		return "", "", nil
	}

	// Validate HCP cluster
	if !aws.IsHostedCP(cluster) {
		return "", "", fmt.Errorf("AutoNode is only supported for Hosted Control Plane clusters")
	}

	currentAutonodeEnabled := ocm.AutoNodeExists(cluster)

	// Validate autonode value if provided
	if autonodeChanged {
		if autonode != "enabled" {
			return "", "", fmt.Errorf("Invalid value for --autonode. Currently only 'enabled' is supported")
		}

		// Check if trying to enable when already enabled
		if currentAutonodeEnabled {
			return "", "", fmt.Errorf("AutoNode is already enabled for this cluster")
		}

		// When enabling AutoNode, IAM role ARN is required
		if !roleArnChanged || roleArn == "" {
			return "", "", fmt.Errorf("IAM role ARN is required when enabling AutoNode")
		}

		autonodeMode = autonode
	}

	// Validate IAM role ARN if provided
	if roleArnChanged {
		if roleArn == "" {
			return "", "", fmt.Errorf("IAM role ARN cannot be empty")
		}

		// Validate ARN format
		if !aws.RoleArnRE.MatchString(roleArn) {
			return "", "", fmt.Errorf("Invalid IAM role ARN format: '%s'. Expected format: arn:aws:iam::<account-id>:role/<role-name>", roleArn)
		}

		// If AutoNode is not being enabled in this request, it must already be enabled
		if !autonodeChanged && !currentAutonodeEnabled {
			return "", "", fmt.Errorf("Cannot update IAM role ARN when AutoNode is not enabled. " +
				"Enable AutoNode first with --autonode=enabled")
		}

		// If AutoNode is already enabled and only updating the role
		if currentAutonodeEnabled && !autonodeChanged {
			autonodeMode = "enabled" // Maintain the current state
		}

		autonodeRole = roleArn
	}

	// Display confirmation prompts based on the changes being made
	if autonodeMode == "enabled" && !currentAutonodeEnabled {
		// Enabling AutoNode for the first time
		r.Reporter.Warnf("You are choosing to enable automatic node management (AutoNode)")
		if !confirm.Confirm("enable AutoNode for cluster with the provided role arn '%s'", autonodeRole) {
			os.Exit(0)
		}
	} else if autonodeRole != "" && currentAutonodeEnabled {
		// Updating the IAM role ARN for existing AutoNode
		currentRoleArn, _ := ocm.GetAutoNodeRoleArn(cluster)
		if currentRoleArn != autonodeRole {
			r.Reporter.Warnf("You are choosing to update the AutoNode IAM role ARN")
			if !confirm.Confirm("update AutoNode IAM role ARN from '%s' to '%s'", currentRoleArn, autonodeRole) {
				os.Exit(0)
			}
		}
	}

	return autonodeMode, autonodeRole, nil
}

// InteractivePrompt handles interactive mode for AutoNode configuration
func InteractivePrompt(r *rosa.Runtime, cmd *cobra.Command, cluster *cmv1.Cluster) (
	autonodeMode string, autonodeRole string, err error) {

	autonodeEnabled := ocm.AutoNodeExists(cluster)
	currentRoleArn, _ := ocm.GetAutoNodeRoleArn(cluster)

	// Build the appropriate question based on current state
	var question string
	if autonodeEnabled {
		question = fmt.Sprintf("Update AutoNode IAM role ARN (current: %s)", currentRoleArn)
	} else {
		question = "Enable automatic node management (AutoNode)"
	}

	// First prompt: enable/update decision
	proceed, err := interactive.GetBool(interactive.Input{
		Question: question,
		Default:  false,
		Required: false,
	})
	if err != nil {
		return "", "", err
	}

	if !proceed {
		return "", "", nil
	}

	// Set mode to enabled
	autonodeMode = "enabled"

	// Build role ARN prompt based on state
	roleQuestion := "AutoNode IAM role ARN"
	if autonodeEnabled {
		roleQuestion = "New AutoNode IAM role ARN"
	}

	// Second prompt: IAM role ARN
	roleArnValue, err := interactive.GetString(interactive.Input{
		Question: roleQuestion,
		Help:     cmd.Flags().Lookup("autonode-iam-role-arn").Usage,
		Default:  currentRoleArn, // Empty string if not enabled
		Required: true,
		Validators: []interactive.Validator{
			interactive.RegExp(aws.RoleArnRE.String()),
		},
	})
	if err != nil {
		return "", "", err
	}
	autonodeRole = strings.TrimSpace(roleArnValue)

	return autonodeMode, autonodeRole, nil
}
