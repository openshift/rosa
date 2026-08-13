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

// ValidateURL returns an error if rawURL cannot be parsed or its scheme is not HTTPS.
func ValidateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid --hyperfleet-url %q: %w", rawURL, err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("--hyperfleet-url must use HTTPS, got %q", rawURL)
	}
	return nil
}

// ExtractRegion parses the AWS region from a Platform API endpoint URL using
// the same approach as the Platform API SDK's rest.Config.ResolveRegion.
func ExtractRegion(rawURL string) (string, error) {
	if err := ValidateURL(rawURL); err != nil {
		return "", err
	}
	u, _ := url.Parse(rawURL) // already validated above
	region := awsRegionRE.FindString(u.Hostname())
	if region == "" {
		return "", fmt.Errorf(
			"cannot derive AWS region from --hyperfleet-url %q; use --region to specify it explicitly",
			rawURL,
		)
	}
	return region, nil
}

// CheckRegionConflict returns an error when an explicitly provided region can be
// compared against the region embedded in rawURL and they differ — a configuration
// error that would cause SigV4 to sign for the wrong endpoint. When the URL
// contains no recognizable region (e.g. a VPN or custom endpoint) a warning is
// emitted instead, since signing with the explicit region may be intentional.
func CheckRegionConflict(explicitRegion, rawURL string, r reporter.Logger) error {
	urlRegion, err := ExtractRegion(rawURL)
	if err != nil {
		r.Warnf("cannot verify region for --hyperfleet-url %s; SigV4 will sign with %s", rawURL, explicitRegion)
		return nil
	}
	if explicitRegion != urlRegion {
		return fmt.Errorf(
			//nolint:lll
			"--region %s does not match region in --hyperfleet-url (%s); use --region %s or omit --region to derive it from the URL",
			explicitRegion,
			urlRegion,
			urlRegion,
		)
	}
	return nil
}
