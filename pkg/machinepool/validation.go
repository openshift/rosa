package machinepool

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2/core"

	"github.com/openshift/rosa/pkg/aws"
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

// validateCapacityReservationReplicas validates that the requested replicas don't exceed available capacity
func validateCapacityReservationReplicas(capacityReservationId string, requestedReplicas int,
	awsClient aws.Client, isAutoscaling bool, minReplicas int, maxReplicas int) error {

	if capacityReservationId == "" {
		return nil // No capacity reservation, no validation needed
	}

	_, availableInstances, err := awsClient.GetCapacityReservationDetails(capacityReservationId)
	if err != nil {
		return fmt.Errorf("unable to validate capacity reservation '%s': %v", capacityReservationId, err)
	}

	if isAutoscaling {
		// For autoscaling, validate both min and max replicas
		if minReplicas > int(availableInstances) {
			return fmt.Errorf("cannot set min replicas to %d: capacity reservation '%s' only has %d available instance(s)",
				minReplicas, capacityReservationId, availableInstances)
		}
		if maxReplicas > int(availableInstances) {
			return fmt.Errorf("cannot set max replicas to %d: capacity reservation '%s' only has %d available instance(s)",
				maxReplicas, capacityReservationId, availableInstances)
		}
	} else {
		// For fixed replicas
		if requestedReplicas > int(availableInstances) {
			return fmt.Errorf("cannot set replicas to %d: capacity reservation '%s' only has %d available instance(s)",
				requestedReplicas, capacityReservationId, availableInstances)
		}
	}

	return nil
}
