package ingress

import (
	"github.com/spf13/pflag"
)

const (
	DefaultIngressRouteSelectorFlag            = "default-ingress-route-selector"
	DefaultIngressExcludedNamespacesFlag       = "default-ingress-excluded-namespaces"
	DefaultIngressWildcardPolicyFlag           = "default-ingress-wildcard-policy"
	DefaultIngressNamespaceOwnershipPolicyFlag = "default-ingress-namespace-ownership-policy"
)

func IsDefaultIngressSetViaCLI(cmd *pflag.FlagSet) bool {
	for _, parameter := range []string{DefaultIngressRouteSelectorFlag,
		DefaultIngressExcludedNamespacesFlag, DefaultIngressWildcardPolicyFlag,
		DefaultIngressNamespaceOwnershipPolicyFlag} {

		if cmd.Changed(parameter) {
			return true
		}
	}

	return false
}
