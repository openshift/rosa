package handler

import (
	"fmt"
	"strings"

	"github.com/openshift/rosa/tests/utils/helper"
)

// AddClusterDomainToNoProxy preserves existing bypass entries and adds the public cluster domain.
func AddClusterDomainToNoProxy(noProxy, clusterName, baseDomain string) string {
	entries := helper.RemoveFromStringSlice(strings.Split(noProxy, ","), "")
	domain := fmt.Sprintf(".%s.%s", clusterName, baseDomain)
	entries = helper.AppendToStringSliceIfNotExist(entries, domain)
	return strings.Join(entries, ",")
}
