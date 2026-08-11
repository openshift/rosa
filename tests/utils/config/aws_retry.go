package config

import (
	"bytes"
	"strings"
	"time"

	"github.com/openshift/rosa/tests/utils/log"
)

// isAWSRegionsCredentialTransient reports whether out/err indicate a transient
// failure retrieving AWS regions (credentials not yet valid for OCM/AWS).
// rosa create cluster calls GetRegionList early; this can flake before the
// intended validation error is reached.
func isAWSRegionsCredentialTransient(out bytes.Buffer, err error) bool {
	combined := out.String()
	if err != nil {
		combined += "\n" + err.Error()
	}
	return strings.Contains(combined, "Failed to retrieve AWS regions") ||
		strings.Contains(combined, "AWS was not able to validate the provided access credentials")
}

// RetryOnAWSRegionsCredentialError retries fn when the result indicates a
// transient AWS regions / credentials failure (for example CLUSTERS-MGMT-400
// while listing regions). Modeled on RetryOnIAMPropagationError.
func RetryOnAWSRegionsCredentialError(
	fn func() (bytes.Buffer, error),
	maxRetries int,
	delay time.Duration,
) (bytes.Buffer, error) {
	var out bytes.Buffer
	var err error
	for i := 0; i <= maxRetries; i++ {
		out, err = fn()
		if !isAWSRegionsCredentialTransient(out, err) {
			return out, err
		}
		if i < maxRetries {
			log.Logger.Infof(
				"AWS regions/credentials not yet available (attempt %d/%d), retrying in %s...",
				i+1, maxRetries+1, delay,
			)
			time.Sleep(delay)
		}
	}
	return out, err
}
