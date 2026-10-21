package public

// ClusterProxy exposes cluster proxy settings using OCM-compatible field names.
// It is populated from spec.hostedCluster.configuration.proxy in responses.
type ClusterProxy struct {
	// HTTPProxy is the URL of the proxy for HTTP requests.
	// +optional
	HTTPProxy string `json:"http_proxy,omitempty"`
	// HTTPSProxy is the URL of the proxy for HTTPS requests.
	// +optional
	HTTPSProxy string `json:"https_proxy,omitempty"`
	// NoProxy lists hosts, domains, IP addresses, or CIDRs excluded from proxying.
	// +optional
	NoProxy string `json:"no_proxy,omitempty"`
}
