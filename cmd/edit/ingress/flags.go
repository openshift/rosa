package ingress

import (
	"github.com/spf13/pflag"
)

const (
	privateFlag    = "private"
	labelMatchFlag = "label-match"
	lbTypeFlag     = "lb-type"

	routeSelectorFlag             = "route-selector"
	excludedNamespacesFlag        = "excluded-namespaces"
	wildcardPolicyFlag            = "wildcard-policy"
	namespaceOwnershipPolicyFlag  = "namespace-ownership-policy"
	clusterRoutesHostnameFlag     = "cluster-routes-hostname"
	clusterRoutesTlsSecretRefFlag = "cluster-routes-tls-secret-ref"
)

var exclusivelyIngressV2Flags = []string{excludedNamespacesFlag, wildcardPolicyFlag,
	namespaceOwnershipPolicyFlag, clusterRoutesHostnameFlag, clusterRoutesTlsSecretRefFlag}

func IsIngressV2SetViaCLI(cmd *pflag.FlagSet) bool {
	for _, parameter := range exclusivelyIngressV2Flags {
		if cmd.Changed(parameter) {
			return true
		}
	}

	return false
}
