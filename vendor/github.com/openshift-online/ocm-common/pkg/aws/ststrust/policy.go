package ststrust

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
)

// Trust policy JSON constants for AssumeRole and ExternalId handling.
const (
	assumeRoleAction     = "sts:AssumeRole"
	externalIDCondition  = "sts:ExternalId"
	policyVersionDefault = "2012-10-17"
)

// PolicyDocument models an IAM trust policy document.
type PolicyDocument struct {
	// Version is the IAM policy version, typically "2012-10-17".
	Version string `json:"Version,omitempty"`
	// Statement holds trust policy statements.
	Statement []PolicyStatement `json:"Statement"`
}

// PolicyStatement models a single IAM policy statement.
type PolicyStatement struct {
	// Sid is the optional statement identifier.
	Sid string `json:"Sid,omitempty"`
	// Effect is typically "Allow" or "Deny".
	Effect string `json:"Effect"`
	// Principal identifies who may assume the role.
	Principal *PolicyStatementPrincipal `json:"Principal,omitempty"`
	// Action is sts:AssumeRole or a list of actions.
	Action interface{} `json:"Action,omitempty"`
	// Resource is the optional resource element.
	Resource interface{} `json:"Resource,omitempty"`
	// Condition holds IAM condition operators such as StringEquals.
	Condition map[string]interface{} `json:"Condition,omitempty"`
}

// PolicyStatementPrincipal models the Principal element in a trust policy statement.
type PolicyStatementPrincipal struct {
	// AWS lists AWS principal ARNs or account identifiers.
	AWS interface{} `json:"AWS,omitempty"`
	// Service lists AWS service principals.
	Service interface{} `json:"Service,omitempty"`
	// Federated is a federated identity provider ARN.
	Federated string `json:"Federated,omitempty"`
}

// parsePolicyDocument decodes and unmarshals a trust policy JSON document.
func parsePolicyDocument(policyJSON string) (*PolicyDocument, error) {
	decoded, err := decodePolicyDocument(policyJSON)
	if err != nil {
		return nil, err
	}
	doc := &PolicyDocument{}
	if err := json.Unmarshal([]byte(decoded), doc); err != nil {
		return nil, fmt.Errorf("failed to parse trust policy JSON: %w", err)
	}
	return doc, nil
}

// marshalPolicyDocument serializes a trust policy document to JSON.
func marshalPolicyDocument(doc *PolicyDocument) (string, error) {
	if doc.Version == "" {
		doc.Version = policyVersionDefault
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("failed to marshal trust policy JSON: %w", err)
	}
	return string(out), nil
}

// decodePolicyDocument percent-decodes policy JSON using PathUnescape.
func decodePolicyDocument(policyJSON string) (string, error) {
	decoded, err := url.PathUnescape(policyJSON)
	if err != nil {
		return "", fmt.Errorf("failed to decode trust policy document: %w", err)
	}
	return decoded, nil
}

// statementAllowsAssumeRole reports whether the statement allows sts:AssumeRole.
func statementAllowsAssumeRole(statement PolicyStatement) bool {
	if statement.Effect != "Allow" {
		return false
	}
	return actionIncludesAssumeRole(statement.Action)
}

// actionIncludesAssumeRole reports whether the Action element includes sts:AssumeRole.
func actionIncludesAssumeRole(action interface{}) bool {
	switch v := action.(type) {
	case string:
		return v == assumeRoleAction
	case []interface{}:
		for _, el := range v {
			if s, ok := el.(string); ok && s == assumeRoleAction {
				return true
			}
		}
	}
	return false
}

// uniqueSorted returns non-empty IDs in sorted order without duplicates.
func uniqueSorted(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		seen[id] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// setUnion returns the sorted union of two ID lists.
func setUnion(a, b []string) []string {
	return uniqueSorted(append(append([]string{}, a...), b...))
}

// setIntersection returns the sorted intersection of two ID lists.
func setIntersection(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	lookup := make(map[string]struct{}, len(a))
	for _, id := range a {
		lookup[id] = struct{}{}
	}
	var out []string
	for _, id := range b {
		if _, ok := lookup[id]; ok {
			out = append(out, id)
		}
	}
	return uniqueSorted(out)
}

// formatIDList formats external ID lists for error messages.
func formatIDList(ids []string) string {
	if len(ids) == 0 {
		return "none"
	}
	return fmt.Sprintf("%v", ids)
}
