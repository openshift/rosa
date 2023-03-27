package machinepools

import (
	"fmt"
	"strconv"

	"github.com/openshift/rosa/pkg/interactive"
)

func MinNodePoolReplicaValidator() interactive.Validator {
	return func(val interface{}) error {
		minReplicas, err := strconv.Atoi(fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}
		if minReplicas < 1 {
			return fmt.Errorf("min-replicas must be greater than zero")
		}
		return nil
	}
}

func MaxNodePoolReplicaValidator(minReplicas int) interactive.Validator {
	return func(val interface{}) error {
		maxReplicas, err := strconv.Atoi(fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}
		if minReplicas > maxReplicas {
			return fmt.Errorf("max-replicas must be greater or equal to min-replicas")
		}
		return nil
	}
}
