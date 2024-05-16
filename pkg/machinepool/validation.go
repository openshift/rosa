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
