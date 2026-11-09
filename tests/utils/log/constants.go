// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"regexp"
)

const (
	RedactValue = "*************"
)

var RedactKeyList = []*regexp.Regexp{
	regexp.MustCompile(`(\\?"password\\?":\\?")([^"]*)(\\?")`),
	regexp.MustCompile(`(\\?"additional_trust_bundle\\?":\\?")([^"]*)(\\?")`),
	regexp.MustCompile(`(-----BEGIN CERTIFICATE-----\n)([^-----]*)(-----END CERTIFICATE-----)`),
	regexp.MustCompile(`(--password\s+)([^\n\t\\\s]+)([\n\\\s]*)`),
	regexp.MustCompile(`(--client-id\s+)([^\n\t\\\s]+)([\n\\\s]*)`),
	regexp.MustCompile(`(--bind-password\s+)([^\n\t\\\s]+)([\n\\\s]*)`),
	regexp.MustCompile(`(--client-secret\s+)([^\n\t\\\s]+)([\n\\\s]*)`),
	regexp.MustCompile(`(--users\s+[a-zA-Z0-9-]+:)([^\s\n\\]+)([\s\S]+)`),
	regexp.MustCompile(`(--cluster-admin-password\s+)([^\n\t\\\s]+)([\n\\\s]*)`),
	regexp.MustCompile(`(--billing-account\s+)([^\n\t\\\s]+)([\n\\\s]*)`),
	regexp.MustCompile(`(arn:aws:[a-z]+:[a-z0-9-]*:)([0-9]{12})(:)`),
	regexp.MustCompile(`("?AWS Account\s?I?D?"?:\s*"?)([0-9]{12})([\n\\n"\s\t,].)`),
	regexp.MustCompile(`(AWS Billing Account:\s*)([0-9]{12})([\n\\n].)`),
	regexp.MustCompile(`("billing_account_id"\:?\s*")([0-9]{12})(")`),
	regexp.MustCompile(`(OCM AWS role on\s*')([0-9]{12})(')`),
	regexp.MustCompile(`(_token\s*)([\w.-]+)([\s\S]*)`),
	// Account IDs in network interface diagnostics, such as owner_id="123456789012".
	// A value containing 12 consecutive digits is replaced entirely; other values,
	// such as service aliases or empty values, are kept.
	regexp.MustCompile("(\\b(?:instance_owner|owner|requester)_id=\")([^\"]*[0-9]{12}[^\"]*)(\")"),
}
