package kubeletconfig

import (
	"fmt"

	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/rosa"
)

type CapabilityChecker interface {
	IsCapabilityEnabled(capability string) (bool, error)
}

// GetMaxPidsLimit - returns the maximum pids limit for the current organization
// the maximum is varied depending on whether the current organizaton has
// the capability.organization.bypass_pids_limit capability
func GetMaxPidsLimit(client CapabilityChecker) (int, error) {
	enabled, err := client.IsCapabilityEnabled(ByPassPidsLimitCapability)
	if err != nil {
		return -1, err
	}

	if enabled {
		return MaxUnsafePodPidsLimit, nil
	}
	return MaxPodPidsLimit, nil
}

func GetInteractiveMaxPidsLimitHelp(maxPidsLimit int) string {
	return fmt.Sprintf(InteractivePodPidsLimitHelp, maxPidsLimit)
}

func GetInteractiveInput(maxPidsLimit int, kubeletConfig *v1.KubeletConfig) interactive.Input {

	var defaultLimit = MinPodPidsLimit
	if kubeletConfig != nil {
		defaultLimit = kubeletConfig.PodPidsLimit()
	}

	return interactive.Input{
		Question: InteractivePodPidsLimitPrompt,
		Help:     GetInteractiveMaxPidsLimitHelp(maxPidsLimit),
		Options:  nil,
		Default:  defaultLimit,
		Required: true,
		Validators: []interactive.Validator{
			interactive.MinValue(MinPodPidsLimit),
			interactive.MaxValue(maxPidsLimit),
		},
	}
}

func PromptForName(requestedName string, forceInteractive bool) (string, error) {

	if requestedName == "" && forceInteractive {
		interactive.Enable()
		return interactive.GetString(interactive.Input{
			Question: InteractiveNameHelpPrompt,
			Help:     InteractiveNameHelp,
			Options:  nil,
			Default:  requestedName,
			Required: true,
		})
	}

	return requestedName, nil
}

// ValidateOrPromptForRequestedPidsLimit validates user provided limits or prompts via interactive mode
// if the user hasn't specified any limit on the command line.
func ValidateOrPromptForRequestedPidsLimit(
	requestedPids int,
	clusterKey string,
	kubeletConfig *v1.KubeletConfig,
	r *rosa.Runtime) (int, error) {

	if requestedPids == PodPidsLimitOptionDefaultValue && !interactive.Enabled() {
		interactive.Enable()
		r.Reporter.Infof("Enabling interactive mode")
	}

	// If the user has provided a fully supported podPidsLimit, then no need to check the org capabilities.
	if requestedPids >= MinPodPidsLimit && requestedPids <= MaxPodPidsLimit {
		return requestedPids, nil
	}

	// The user is trying to exceed the default ranges. This requires a capability check at the org-level
	maxPidsLimit, err := GetMaxPidsLimit(r.OCMClient)
	if err != nil {
		return PodPidsLimitOptionDefaultValue,
			fmt.Errorf("Failed to check maximum allowed Pids limit for cluster '%s'",
				clusterKey)
	}

	if interactive.Enabled() {
		requestedPids, err = interactive.GetInt(GetInteractiveInput(maxPidsLimit, kubeletConfig))

		if err != nil {
			return PodPidsLimitOptionDefaultValue,
				fmt.Errorf("Failed reading requested Pids limit for cluster '%s': '%s'",
					clusterKey, err)
		}
	}

	if requestedPids < MinPodPidsLimit {
		return PodPidsLimitOptionDefaultValue,
			fmt.Errorf("The minimum value for --pod-pids-limit is '%d'. You have supplied '%d'",
				MinPodPidsLimit, requestedPids)
	}

	if requestedPids > maxPidsLimit {
		return PodPidsLimitOptionDefaultValue,
			fmt.Errorf("The maximum value for --pod-pids-limit is '%d'. You have supplied '%d'",
				maxPidsLimit, requestedPids)
	}

	return requestedPids, nil
}
