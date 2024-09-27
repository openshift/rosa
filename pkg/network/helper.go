package network

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cfTypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/sirupsen/logrus"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
)

// logStackEvents fetches and logs stack events
func logStackEvents(cfClient *cloudformation.Client, stackName string, logger *logrus.Logger) {
	events, err := cfClient.DescribeStackEvents(context.TODO(), &cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		logger.Errorf("Failed to describe stack events: %v", err)
		return
	}

	// Group events by resource and keep the latest event for each resource
	latestEvents := make(map[string]cfTypes.StackEvent)
	for _, event := range events.StackEvents {
		resource := aws.ToString(event.LogicalResourceId)
		if existingEvent, exists := latestEvents[resource]; !exists || event.Timestamp.After(*existingEvent.Timestamp) {
			latestEvents[resource] = event
		}
	}

	logger.Info("---------------------------------------------")
	// Log the latest event for each resource with color
	for resource, event := range latestEvents {
		statusColor := getStatusColor(event.ResourceStatus)
		reason := aws.ToString(event.ResourceStatusReason)

		readableReason := strings.ReplaceAll(reason, ". ", ".\n    ")

		// Check for "Access Denied" in the reason
		if strings.Contains(reason, "AccessDenied") {
			logger.Warnf("Resource: %s, Status: %s%s%s, Reason: %s (Access Denied)",
				resource, statusColor, event.ResourceStatus, ColorReset, readableReason)
		} else {
			logger.Infof("Resource: %s, Status: %s%s%s, Reason: %s",
				resource, statusColor, event.ResourceStatus, ColorReset, readableReason)
		}
	}
}

func getStatusColor(status cfTypes.ResourceStatus) string {
	switch status {
	case cfTypes.ResourceStatusCreateComplete, cfTypes.ResourceStatusUpdateComplete:
		return ColorGreen
	case cfTypes.ResourceStatusCreateFailed, cfTypes.ResourceStatusDeleteFailed, cfTypes.ResourceStatusUpdateFailed:
		return ColorRed
	default:
		return ColorYellow
	}
}

// parseParams converts the list of parameter strings into a map and sets default values
func ParseParams(params []string) (map[string]string, map[string]string) {
	result := make(map[string]string)
	userTags := map[string]string{}

	for _, param := range params {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) == 2 {
			if parts[0] == "Tags" {
				tagEntries := strings.Split(parts[1], ",")
				for _, entry := range tagEntries {
					tagParts := strings.SplitN(entry, "=", 2)
					if len(tagParts) == 2 {
						userTags[tagParts[0]] = tagParts[1]
					}
				}
			} else {
				result[parts[0]] = parts[1]
			}
		}
	}

	return result, userTags
}

// selectTemplate selects the appropriate template file based on the template name
func SelectTemplate(command string) string {
	return fmt.Sprintf("cmd/create/network/templates/%s/cloudformation.yaml", command)
}
