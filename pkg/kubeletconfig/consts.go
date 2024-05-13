package kubeletconfig

const (
	MinPodPidsLimit                = 4096
	MaxPodPidsLimit                = 16384
	MaxUnsafePodPidsLimit          = 3694303
	NameOption                     = "name"
	NameOptionDefaultValue         = ""
	NameOptionUsage                = "Sets the name for this KubeletConfig (optional, generated if omitted)"
	PodPidsLimitOption             = "pod-pids-limit"
	PodPidsLimitOptionUsage        = "Sets the requested pod_pids_limit for this KubeletConfig."
	PodPidsLimitOptionDefaultValue = 0
	InteractivePodPidsLimitPrompt  = "Pod Pids Limit?"
	InteractivePodPidsLimitHelp    = "Set the Pod Pids Limit field to a value between 4096 and %d"
	InteractiveNameHelpPrompt      = "Name?"
	InteractiveNameHelp            = "Set the name of this KubeletConfig (optional)"
	ByPassPidsLimitCapability      = "capability.organization.bypass_pids_limits"
)
