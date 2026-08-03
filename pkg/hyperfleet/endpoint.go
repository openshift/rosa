package hyperfleet

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/openshift/rosa/pkg/reporter"
)

// awsRegionRE matches standard and GovCloud AWS region names within a URL hostname.
// Examples: us-east-1, ap-southeast-1, us-gov-east-1
var awsRegionRE = regexp.MustCompile(`[a-z]+-(?:[a-z]+-)+\d+`)

// ExtractRegion parses the AWS region from a Platform API endpoint URL using
// the same approach as the Platform API SDK's rest.Config.ResolveRegion.
func ExtractRegion(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid --hyperfleet-url %q: %w", rawURL, err)
	}
	region := awsRegionRE.FindString(u.Hostname())
	if region == "" {
		return "", fmt.Errorf(
			"cannot derive AWS region from --hyperfleet-url %q; use --region to specify it explicitly",
			rawURL,
		)
	}
	return region, nil
}

// WarnOnMismatch emits a warning when an explicitly provided --region does not
// match the region embedded in --hyperfleet-url, or when the URL contains no
// recognizable region at all.
func WarnOnMismatch(explicitRegion, rawURL string, r reporter.Logger) {
	urlRegion, err := ExtractRegion(rawURL)
	if err != nil {
		r.Warnf("cannot verify region for --hyperfleet-url %s; SigV4 will sign with %s", rawURL, explicitRegion)
		return
	}
	if explicitRegion != urlRegion {
		r.Warnf(
			"resolved region %s does not match region in --hyperfleet-url %s; SigV4 will sign with %s",
			explicitRegion, rawURL, explicitRegion,
		)
	}
}
