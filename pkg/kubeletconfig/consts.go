package kubeletconfig

const (
	MinPodPidsLimit                = 4096
	MaxPodPidsLimit                = 16384
	MaxUnsafePodPidsLimit          = 3694303
	NameOption                     = "name"
	NameOptionDefaultValue         = ""
	NameOptionUsage                = "Name of the KubeletConfig (required for Hosted Control Plane clusters)"
	PodPidsLimitOption             = "pod-pids-limit"
	PodPidsLimitOptionUsage        = "Sets the requested pod_pids_limit for this KubeletConfig."
	PodPidsLimitOptionDefaultValue = 0
	InteractivePodPidsLimitPrompt  = "Pod Pids Limit?"
	InteractivePodPidsLimitHelp    = "Set the Pod Pids Limit field to a value between 4096 and %d"
	InteractiveNameHelpPrompt      = "Name?"
	InteractiveNameHelp            = "Name of the KubeletConfig"
	ByPassPidsLimitCapability      = "capability.organization.bypass_pids_limits"
)
