package ingress

import (
	"strings"
)

func GetExcludedNamespaces(excludedNamespaces string) []string {
	if excludedNamespaces == "" {
		return []string{}
	}
	sliceExcludedNamespaces := strings.Split(excludedNamespaces, ",")
	for i := range sliceExcludedNamespaces {
		sliceExcludedNamespaces[i] = strings.TrimSpace(sliceExcludedNamespaces[i])
	}
	return sliceExcludedNamespaces
}
