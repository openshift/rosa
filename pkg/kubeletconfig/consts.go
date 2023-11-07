package kubeletconfig

const (
	MinPodPidsLimit         = 4096
	PodPidsLimitOption      = "pod-pids-limit"
	PodPidsLimitOptionUsage = "Sets the requested pod_pids_limit for your custom KubeletConfig." +
		" Must be an integer in the range 4096 - 16,384."
	PodPidsLimitOptionDefaultValue = -1

	InteractivePodPidsLimitPrompt = "Pod Pids Limit?"
	InteractivePodPidsLimitHelp   = "Set the Pod Pids Limit field to a value between 4096 and 16,384"
)
