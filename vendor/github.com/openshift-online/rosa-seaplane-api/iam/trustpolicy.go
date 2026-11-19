package iam

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Allow-lists for values interpolated into the trust-policy JSON. The trust
// policy is a security-sensitive IAM document, so inputs are validated with
// allow-lists at the trust boundary rather than escaped after the fact.
var (
	partitionRe = regexp.MustCompile(`^aws(-[a-z]+)*$`)
	accountIDRe = regexp.MustCompile(`^[0-9]{12}$`)
	// issuerDomainRe accepts RFC 3986 pchar in the path (unreserved /
	// pct-encoded / sub-delims / ":" / "@"), excluding only the JSON-unsafe
	// characters " and \", which are the two that could break out of the JSON
	// string the domain is interpolated into. Real OIDC issuers (AWS, Azure AD,
	// multi-tenant IdPs) routinely use '=', '@', '+', etc. in paths.
	issuerDomainRe = regexp.MustCompile(`^[A-Za-z0-9.-]+(:[0-9]{1,5})?(/[A-Za-z0-9._~!$&'()*+,;=:@/-]*)?$`)
)

// validateAWSInputs checks the AWS identity values that get interpolated
// into generated IAM documents and ARNs.
func validateAWSInputs(partition, accountID string) error {
	if !partitionRe.MatchString(partition) {
		return fmt.Errorf("invalid AWS partition %q", partition)
	}
	if !accountIDRe.MatchString(accountID) {
		return fmt.Errorf("invalid AWS account ID %q", accountID)
	}
	return nil
}

// BuildEC2TrustPolicy returns the trust policy for the worker node instance
// role, which is trusted by the EC2 service (not OIDC web identity).
func BuildEC2TrustPolicy() string {
	return `{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "Service": ["ec2.amazonaws.com"]
    },
    "Action": "sts:AssumeRole"
  }]
}`
}

// BuildWebIdentityTrustPolicy returns the OIDC web-identity trust policy
// for an operator role, conditioned on the role's service accounts (sub)
// and the "openshift" audience. No client should hand-write this JSON.
func BuildWebIdentityTrustPolicy(input TrustPolicyInput, role OperatorRoleSpec) (string, error) {
	if role.IsWorkerRole {
		return "", fmt.Errorf("role %q is the worker role; use BuildEC2TrustPolicy", role.Name)
	}
	if len(role.ServiceAccounts) == 0 {
		return "", fmt.Errorf("role %q has no service accounts", role.Name)
	}
	if err := validateAWSInputs(input.Partition, input.AccountID); err != nil {
		return "", err
	}

	issuerDomain, err := IssuerDomain(input.IssuerURL)
	if err != nil {
		return "", err
	}
	if !issuerDomainRe.MatchString(issuerDomain) {
		return "", fmt.Errorf("OIDC issuer %q contains unsupported characters", input.IssuerURL)
	}

	subject, err := OIDCSubject(role.ServiceAccounts)
	if err != nil {
		return "", fmt.Errorf("building trust policy for role %q: %w", role.Name, err)
	}

	return fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "Federated": "arn:%s:iam::%s:oidc-provider/%s"
    },
    "Action": "sts:AssumeRoleWithWebIdentity",
    "Condition": {
      "StringEquals": {
        "%s:sub": %s,
        "%s:aud": "openshift"
      }
    }
  }]
}`,
		input.Partition,
		input.AccountID,
		issuerDomain,
		issuerDomain,
		subject,
		issuerDomain,
	), nil
}

// OIDCSubject renders the trust-policy "sub" condition value: a JSON string
// for a single service account, a JSON array for multiple — matching AWS
// StringEquals semantics and the V1/V2 CLI behavior.
func OIDCSubject(serviceAccounts []string) (string, error) {
	if len(serviceAccounts) == 0 {
		return "", fmt.Errorf("no service accounts configured")
	}
	var subject any
	if len(serviceAccounts) == 1 {
		subject = serviceAccounts[0]
	} else {
		subject = serviceAccounts
	}
	encoded, err := json.Marshal(subject)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// IssuerDomain derives the OIDC issuer domain (host + path, without the
// scheme) from the issuer URL. This matches how the AWS OIDC provider is
// registered, so the trust policy always references the provider that was
// actually created. A naive scheme-strip would diverge whenever the issuer
// carries a query string, fragment, or trailing slash.
func IssuerDomain(issuerURL string) (string, error) {
	parsed, err := url.ParseRequestURI(issuerURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "https" {
		return "", fmt.Errorf("expected OIDC issuer URL '%s' to use an https:// scheme", issuerURL)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("OIDC issuer URL '%s' has no host", issuerURL)
	}
	// Use EscapedPath(), not Path: url.Parse decodes percent-encoded
	// characters in .Path, which would make the trust policy reference a
	// decoded path that doesn't match the OIDC provider registered with AWS.
	return parsed.Host + parsed.EscapedPath(), nil
}

// ManagedPolicyARNs renders the full policy ARNs for a role in the given
// AWS partition, e.g. "arn:aws:iam::aws:policy/service-role/ROSAIngressOperatorPolicy".
// Definitions store partition-relative policy paths for this reason.
// A pure formatter: see RoleARN for the validation boundary note.
func ManagedPolicyARNs(role OperatorRoleSpec, partition string) []string {
	arns := make([]string, len(role.ManagedPolicies))
	for i, path := range role.ManagedPolicies {
		arns[i] = fmt.Sprintf("arn:%s:iam::aws:policy/%s", partition, strings.TrimPrefix(path, "/"))
	}
	return arns
}
