package ingress

import (
	"fmt"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/pflag"

	"github.com/openshift/rosa/pkg/helper"
	. "github.com/openshift/rosa/pkg/ingress"
)

type stringTransformation func(source string) string

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
	componentRoutesFlag           = "component-routes"

	expectedLengthOfParsedComponent = 2
	hostnameParameter               = "hostname"
	//nolint:gosec
	tlsSecretRefParameter = "tlsSecretRef"
)

var exclusivelyIngressV2Flags = []string{excludedNamespacesFlag, wildcardPolicyFlag,
	namespaceOwnershipPolicyFlag, clusterRoutesHostnameFlag, clusterRoutesTlsSecretRefFlag, componentRoutesFlag}

var expectedComponentRoutes = []string{
	string(cmv1.ComponentRouteTypeOauth),
	string(cmv1.ComponentRouteTypeConsole),
	string(cmv1.ComponentRouteTypeDownloads),
}

var expectedParameters = []string{
	hostnameParameter,
	tlsSecretRefParameter,
}

func IsIngressV2SetViaCLI(flags *pflag.FlagSet) bool {
	for _, parameter := range exclusivelyIngressV2Flags {
		if flags.Changed(parameter) {
			return true
		}
	}

	return false
}

func addIngressV2Flags(flags *pflag.FlagSet) {
	flags.StringVar(
		&args.excludedNamespaces,
		excludedNamespacesFlag,
		"",
		"Excluded namespaces for ingress. Format should be a comma-separated list 'value1, value2...'. "+
			"If no values are specified, all namespaces will be exposed.",
	)

	flags.StringVar(
		&args.wildcardPolicy,
		wildcardPolicyFlag,
		"",
		fmt.Sprintf("Wildcard Policy for ingress. Options are %s. Default is '%s'.",
			strings.Join(ValidWildcardPolicies, ","), DefaultWildcardPolicy),
	)

	flags.StringVar(
		&args.namespaceOwnershipPolicy,
		namespaceOwnershipPolicyFlag,
		"",
		fmt.Sprintf("Namespace Ownership Policy for ingress. Options are %s. Default is '%s'.",
			strings.Join(ValidNamespaceOwnershipPolicies, ","), DefaultNamespaceOwnershipPolicy),
	)

	flags.StringVar(
		&args.componentRoutes,
		componentRoutesFlag,
		"",
		//nolint:lll
		"Component routes settings. Available keys [oauth, console, downloads]. For each key a pair of hostname and tlsSecretRef is expected to be supplied. "+
			"Format should be a comma separate list 'oauth: hostname=example-hostname;tlsSecretRef=example-secret-ref,downloads:...",
	)
}

func parseComponentRoutes(input string) (map[string]*cmv1.ComponentRouteBuilder, error) {
	result := map[string]*cmv1.ComponentRouteBuilder{}
	input = strings.TrimSpace(input)
	components := strings.Split(input, ",")
	if len(components) != len(expectedComponentRoutes) {
		return nil, fmt.Errorf(
			"the expected amount of component routes is %d, but %d have been supplied",
			len(expectedComponentRoutes),
			len(components),
		)
	}
	transformations := []stringTransformation{
		func(source string) string {
			return strings.TrimSpace(source)
		},
		func(source string) string {
			return strings.Trim(source, "\"")
		},
	}
	for _, component := range components {
		component = strings.TrimSpace(component)
		parsedComponent := strings.Split(component, ":")
		if len(parsedComponent) != expectedLengthOfParsedComponent {
			return nil, fmt.Errorf(
				"only the name of the component should be followed by ':' " +
					"or the component should always include it's parameters separated by ':'",
			)
		}
		componentName := strings.TrimSpace(parsedComponent[0])
		if !helper.Contains(expectedComponentRoutes, componentName) {
			return nil, fmt.Errorf(
				"'%s' is not a valid component name. Expected include %s",
				componentName,
				helper.SliceToSortedString(expectedComponentRoutes),
			)
		}
		parameters := strings.TrimSpace(parsedComponent[1])
		componentRouteBuilder := new(cmv1.ComponentRouteBuilder)
		parsedParameter := strings.Split(parameters, ";")
		if len(parsedParameter) != len(expectedParameters) {
			return nil, fmt.Errorf(
				"only %d parameters are expected for each component",
				len(expectedParameters),
			)
		}
		for _, values := range parsedParameter {
			values = strings.TrimSpace(values)
			parsedValues := strings.Split(values, "=")
			if len(parsedValues) != expectedLengthOfParsedComponent {
				return nil, fmt.Errorf(
					"only the name of the parameter should be followed by '=' " +
						"or the paremater should always include a value separated by '='",
				)
			}
			parameterName := strings.TrimSpace(parsedValues[0])
			if !helper.Contains(expectedParameters, parameterName) {
				return nil, fmt.Errorf(
					"'%s' is not a valid parameter for a component route. Expected include %s",
					parameterName,
					helper.SliceToSortedString(expectedParameters),
				)
			}
			parameterValue := parsedValues[1]
			for _, t := range transformations {
				parameterValue = t(parameterValue)
			}
			// TODO: use reflection, couldn't get it to work
			if parameterName == hostnameParameter {
				componentRouteBuilder.Hostname(parameterValue)
			} else if parameterName == tlsSecretRefParameter {
				componentRouteBuilder.TlsSecretRef(parameterValue)
			}
		}
		result[componentName] = componentRouteBuilder
	}
	return result, nil
}
