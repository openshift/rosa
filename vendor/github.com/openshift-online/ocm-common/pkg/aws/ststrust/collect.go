package ststrust

import (
	"fmt"
)

// CollectSTSExternalIDsFromTrustPolicy returns unique sts:ExternalId values from Allow statements
// that include sts:AssumeRole. Values come from StringEquals and StringEqualsIfExists conditions.
func CollectSTSExternalIDsFromTrustPolicy(policyJSON string) ([]string, error) {
	if policyJSON == "" {
		return nil, nil
	}
	doc, err := parsePolicyDocument(policyJSON)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, statement := range doc.Statement {
		if !statementAllowsAssumeRole(statement) {
			continue
		}
		ids = append(ids, collectExternalIDsFromCondition(statement.Condition)...)
	}
	return uniqueSorted(ids), nil
}

// ExternalIDMatchesTrustPolicy reports whether entered appears in the trust policy ExternalId set.
func ExternalIDMatchesTrustPolicy(entered, policyJSON string) (bool, error) {
	if entered == "" {
		return false, ErrExternalIDEmpty
	}
	ids, err := CollectSTSExternalIDsFromTrustPolicy(policyJSON)
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		if id == entered {
			return true, nil
		}
	}
	return false, nil
}

// collectExternalIDsFromCondition extracts sts:ExternalId values from IAM Condition blocks.
func collectExternalIDsFromCondition(condition map[string]interface{}) []string {
	if condition == nil {
		return nil
	}
	var ids []string
	for _, key := range []string{"StringEquals", "StringEqualsIfExists"} {
		raw, ok := condition[key]
		if !ok {
			continue
		}
		ids = append(ids, externalIDsFromConditionBlock(raw)...)
	}
	return ids
}

// externalIDsFromConditionBlock extracts sts:ExternalId from a single condition operator value.
func externalIDsFromConditionBlock(raw interface{}) []string {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	value, ok := m[externalIDCondition]
	if !ok {
		return nil
	}
	return externalIDsFromValue(value)
}

// externalIDsFromValue normalizes string or array ExternalId condition values.
func externalIDsFromValue(value interface{}) []string {
	switch v := value.(type) {
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	case []interface{}:
		var ids []string
		for _, el := range v {
			if s, ok := el.(string); ok && s != "" {
				ids = append(ids, s)
			}
		}
		return ids
	default:
		return nil
	}
}

// CanInjectSTSExternalID checks whether entered may be applied to an existing trust policy.
// Injection is allowed when the policy has no ExternalId conditions or entered is already listed.
func CanInjectSTSExternalID(existingPolicyJSON, entered string) error {
	if entered == "" {
		return ErrExternalIDEmpty
	}
	ids, err := CollectSTSExternalIDsFromTrustPolicy(existingPolicyJSON)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	for _, id := range ids {
		if id == entered {
			return nil
		}
	}
	return fmt.Errorf("%w: existing trust policy defines %s", ErrExternalIDConflictOnInject, formatIDList(ids))
}
