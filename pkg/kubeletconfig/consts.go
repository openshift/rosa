package kubeletconfig

const (
	MinPodPidsLimit                = 4096
	MaxPodPidsLimit                = 16384
	MaxUnsafePodPidsLimit          = 3694303
	PodPidsLimitOption             = "pod-pids-limit"
	PodPidsLimitOptionUsage        = "Sets the requested pod_pids_limit for your custom KubeletConfig."
	PodPidsLimitOptionDefaultValue = -1
	InteractivePodPidsLimitPrompt  = "Pod Pids Limit?"
	InteractivePodPidsLimitHelp    = "Set the Pod Pids Limit field to a value between 4096 and %d"
	ByPassPidsLimitCapability      = "capability.organization.bypass_pids_limits"
)
