package machinepool

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2/core"
)

func ValidateKubeletConfig(input interface{}) error {
	if strings, ok := input.([]string); ok {
		return validateCount(strings)
	} else if answers, ok := input.([]core.OptionAnswer); ok {
		return validateCount(answers)
	}

	return fmt.Errorf("Input for kubelet config flag is not valid")
}

func validateCount[K any](kubeletConfigs []K) error {
	if len(kubeletConfigs) > 1 {
		return fmt.Errorf("Only a single kubelet config is supported for Machine Pools")
	}
	return nil
}

func validateEditInput(poolType string, autoscaling bool, minReplicas int, maxReplicas int, replicas int,
	isReplicasSet bool, isAutoscalingSet bool, isMinReplicasSet bool, isMaxReplicasSet bool, id string) error {

	if autoscaling && minReplicas < 0 && isMinReplicasSet {
		return fmt.Errorf("Min replicas must be a non-negative number when autoscaling is set")
	}

	if autoscaling && maxReplicas < 0 && isMaxReplicasSet {
		return fmt.Errorf("Max replicas must be a non-negative number when autoscaling is set")
	}

	if !autoscaling && replicas < 0 {
		return fmt.Errorf("Replicas must be a non-negative number")
	}

	if autoscaling && isReplicasSet && isAutoscalingSet {
		return fmt.Errorf("Autoscaling enabled on %s pool '%s'. can't set replicas", poolType, id)
	}

	if autoscaling && isAutoscalingSet && maxReplicas < minReplicas {
		return fmt.Errorf("Max replicas must not be greater than min replicas when autoscaling is enabled")
	}

	if !autoscaling && (isMinReplicasSet || isMaxReplicasSet) {
		return fmt.Errorf("Autoscaling disabled on %s pool '%s'. can't set min or max replicas", poolType, id)
	}

	return nil
}

func validateCapacityReservationId(proposedId, nodepoolId, existingId string) error {
	if existingId != "" {
		return fmt.Errorf("Unable to change 'capacity-reservation-id' to '%s'. AWS NodePool '%s' already has a "+
			"Capacity Reservation ID: '%s'", proposedId, nodepoolId, existingId)
	}
	return nil
}
