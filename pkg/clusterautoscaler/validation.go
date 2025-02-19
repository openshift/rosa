package clusterautoscaler

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/rosa"
)

const (
	ClusterNotReadyMessage = "Cluster '%s' is not yet ready. Current state is '%s'"
	HcpError               = "Unable to use flag '%s' when editing a Hosted Control Plane cluster autoscaler.\n" +
		"Supported flags are: '%s', '%s', '%s', '%s'"
)

func IsAutoscalerSupported(runtime *rosa.Runtime, cluster *cmv1.Cluster) error {

	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf(ClusterNotReadyMessage, runtime.ClusterKey, cluster.State())
	}

	return nil
}

func ValidateAutoscalerFlagsForHostedCp(prefix string, cmd *cobra.Command) (bool, error) {

	flagsToCheck := []string{
		balanceSimilarNodeGroupsFlag,
		skipNodesWithLocalStorageFlag,
		ignoreDaemonsetsUtilizationFlag,
		balancingIgnoredLabelsFlag,
		minCoresFlag,
		maxCoresFlag,
		minMemoryFlag,
		maxMemoryFlag,
		gpuLimitFlag,
		scaleDownEnabledFlag,
		scaleDownUnneededTimeFlag,
		scaleDownUtilizationThresholdFlag,
		scaleDownDelayAfterAddFlag,
		scaleDownDelayAfterDeleteFlag,
		scaleDownDelayAfterFailureFlag,
		logVerbosityFlag,
	}

	for _, flag := range flagsToCheck {
		err := hostedCpValidationHelper(prefix, cmd, flag)
		if err != nil {
			return false, err
		}
	}

	mustHaveAtLeastOneList := []string{
		maxNodesTotalFlag,
		podPriorityThresholdFlag,
		maxPodGracePeriodFlag,
		maxNodeProvisionTimeFlag,
	}

	atLeastOneChanged := false
	for _, flag := range mustHaveAtLeastOneList {
		if cmd.Flag(fmt.Sprintf("%s%s", prefix, flag)).Changed {
			atLeastOneChanged = true
		}
	}

	if !atLeastOneChanged {
		return false, errors.UserErrorf("Must supply at least one of the following flags: '%s', '%s', '%s', '%s'."+
			" Editing a Hosted Control Plane cluster autoscaler does not support interactive mode.", maxNodesTotalFlag,
			podPriorityThresholdFlag, maxPodGracePeriodFlag, maxNodeProvisionTimeFlag)
	}

	return true, nil
}

func hostedCpValidationHelper(prefix string, cmd *cobra.Command, flagName string) error {
	flag := fmt.Sprintf("%s%s", prefix, flagName)
	if cmd.Flag(flag).Changed {
		return errors.UserErrorf(HcpError, flagName,
			maxNodesTotalFlag, maxPodGracePeriodFlag, maxNodeProvisionTimeFlag,
			podPriorityThresholdFlag)
	}
	return nil
}
