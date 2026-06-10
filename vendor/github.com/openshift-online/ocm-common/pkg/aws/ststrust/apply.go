package ststrust

import (
	"fmt"
)

// ApplySTSExternalIDToTrustPolicy adds or updates sts:ExternalId on Allow sts:AssumeRole statements.
// If entered is already present in the policy, the document is returned unchanged.
// If the policy already defines other ExternalIds and entered is not among them, injection fails.
func ApplySTSExternalIDToTrustPolicy(policyJSON, entered string) (string, error) {
	if entered == "" {
		return "", ErrExternalIDEmpty
	}
	if err := ValidateSTSExternalIDFormat(entered); err != nil {
		return "", err
	}
	if policyJSON == "" {
		return "", fmt.Errorf("trust policy document is empty")
	}
	if err := CanInjectSTSExternalID(policyJSON, entered); err != nil {
		return "", err
	}
	ids, err := CollectSTSExternalIDsFromTrustPolicy(policyJSON)
	if err != nil {
		return "", err
	}
	for _, id := range ids {
		if id == entered {
			return policyJSON, nil
		}
	}
	doc, err := parsePolicyDocument(policyJSON)
	if err != nil {
		return "", err
	}
	updated := false
	for i := range doc.Statement {
		if !statementAllowsAssumeRole(doc.Statement[i]) {
			continue
		}
		setExternalIDCondition(&doc.Statement[i], entered)
		updated = true
	}
	if !updated {
		return policyJSON, nil
	}
	return marshalPolicyDocument(doc)
}

// setExternalIDCondition sets sts:ExternalId on the statement StringEquals block.
func setExternalIDCondition(statement *PolicyStatement, externalID string) {
	if statement.Condition == nil {
		statement.Condition = map[string]interface{}{}
	}
	stringEquals, ok := statement.Condition["StringEquals"].(map[string]interface{})
	if !ok || stringEquals == nil {
		stringEquals = map[string]interface{}{}
	}
	stringEquals[externalIDCondition] = externalID
	statement.Condition["StringEquals"] = stringEquals
}
