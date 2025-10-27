package helper

import (
	"bytes"
	"fmt"
	"regexp"
)

// Extract the DNS domain ID from the output of `rosa create dns-domain`
func ExtractDNSDomainID(output bytes.Buffer) (string, error) {
	outputStr := output.String()
	re := regexp.MustCompile(`DNS domain ‘([^’]+)’ has been created`)
	matches := re.FindStringSubmatch(outputStr)

	if len(matches) < 2 {
		return "", fmt.Errorf("failed to extract dns-domain id from the output %s", outputStr)
	}
	return matches[1], nil
}
