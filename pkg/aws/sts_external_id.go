package aws

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/openshift-online/ocm-common/pkg/aws/ststrust"
	accountroles "github.com/openshift-online/ocm-common/pkg/rosa/accountroles"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// RoleTrustPolicyJSON returns the decoded assume-role policy document for an IAM role.
func RoleTrustPolicyJSON(role iamtypes.Role) (string, error) {
	if role.AssumeRolePolicyDocument == nil {
		return "", fmt.Errorf("role '%s' has no assume role policy document", aws.ToString(role.RoleName))
	}
	decoded, err := url.QueryUnescape(aws.ToString(role.AssumeRolePolicyDocument))
	if err != nil {
		return "", fmt.Errorf("failed to decode trust policy for role '%s': %w", aws.ToString(role.RoleName), err)
	}
	return decoded, nil
}

// TrustPolicyJSONForRoleARN fetches and decodes the trust policy for a role ARN.
func TrustPolicyJSONForRoleARN(client Client, roleARN string) (string, error) {
	if roleARN == "" {
		return "", nil
	}
	role, err := client.GetRoleByARN(roleARN)
	if err != nil {
		return "", err
	}
	return RoleTrustPolicyJSON(role)
}

// BuildAccountRoleAssumeRolePolicy builds an account-role trust policy from OCM templates.
// External ID is injected into installer and support trust policies when externalID is non-empty.
func BuildAccountRoleAssumeRolePolicy(
	roleKey, partition string,
	policies map[string]*cmv1.AWSSTSPolicy,
	env, externalID string,
) (string, error) {
	filename := fmt.Sprintf("sts_%s_trust_policy", roleKey)
	policyDetail := GetPolicyDetails(policies, filename)
	base := InterpolatePolicyDocument(partition, policyDetail, map[string]string{
		"partition":      partition,
		"aws_account_id": GetJumpAccount(env),
	})
	if externalID == "" || !accountroles.RequiresSTSExternalIDInTrustPolicy(roleKey) {
		return base, nil
	}
	if base == "" {
		return "", nil
	}
	return ststrust.ApplySTSExternalIDToTrustPolicy(base, externalID)
}

func validateExistingAccountRoleExternalIDPolicy(roleName, existingPolicy, externalID string) error {
	ids, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(existingPolicy)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return fmt.Errorf(
			"existing role '%s' has no sts:ExternalId; delete and recreate the role with --external-id instead of updating it",
			roleName,
		)
	}
	return ststrust.CanInjectSTSExternalID(existingPolicy, externalID)
}

// ValidateExistingAccountRoleExternalID ensures --external-id is compatible with an existing role
// without modifying its trust policy.
func ValidateExistingAccountRoleExternalID(client Client, roleName, roleKey, externalID string) error {
	if externalID == "" || !accountroles.RequiresSTSExternalIDInTrustPolicy(roleKey) {
		return nil
	}
	role, err := client.GetRoleByName(roleName)
	if err != nil {
		return err
	}
	existingPolicy, err := RoleTrustPolicyJSON(role)
	if err != nil {
		return err
	}
	return validateExistingAccountRoleExternalIDPolicy(roleName, existingPolicy, externalID)
}

// ResolveAccountRoleTrustPolicyExternalID validates a requested external ID against an existing role
// and returns the external ID to embed when rebuilding its trust policy.
func ResolveAccountRoleTrustPolicyExternalID(
	client Client, roleName, roleKey, requestedExternalID string,
) (effectiveExternalID, existingTrustPolicy string, err error) {
	if !accountroles.RequiresSTSExternalIDInTrustPolicy(roleKey) {
		return requestedExternalID, "", nil
	}
	role, err := client.GetRoleByName(roleName)
	if err != nil {
		return "", "", err
	}
	existingTrustPolicy, err = RoleTrustPolicyJSON(role)
	if err != nil {
		return "", "", err
	}
	if requestedExternalID != "" {
		err = validateExistingAccountRoleExternalIDPolicy(roleName, existingTrustPolicy, requestedExternalID)
		if err != nil {
			return "", "", err
		}
		return requestedExternalID, existingTrustPolicy, nil
	}
	ids, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(existingTrustPolicy)
	if err != nil {
		return "", "", err
	}
	if len(ids) == 1 {
		return ids[0], existingTrustPolicy, nil
	}
	return "", existingTrustPolicy, nil
}

// PreserveSTSExternalIDsInTrustPolicy copies sts:ExternalId conditions from an existing trust policy
// onto a rebuilt trust policy document.
func PreserveSTSExternalIDsInTrustPolicy(builtPolicy, existingPolicy string) (string, error) {
	ids, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(existingPolicy)
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return builtPolicy, nil
	}
	if len(ids) == 1 {
		return ststrust.ApplySTSExternalIDToTrustPolicy(builtPolicy, ids[0])
	}
	return copySTSExternalIDConditions(builtPolicy, existingPolicy)
}

func copySTSExternalIDConditions(builtPolicy, existingPolicy string) (string, error) {
	condition, err := stsExternalIDConditionFromTrustPolicy(existingPolicy)
	if err != nil || condition == nil {
		return builtPolicy, err
	}

	var builtDoc map[string]interface{}
	if err := json.Unmarshal([]byte(builtPolicy), &builtDoc); err != nil {
		return "", fmt.Errorf("failed to parse rebuilt trust policy: %w", err)
	}
	statements, ok := builtDoc["Statement"].([]interface{})
	if !ok {
		return builtPolicy, nil
	}
	for _, raw := range statements {
		statement, ok := raw.(map[string]interface{})
		if !ok || !trustPolicyStatementAllowsAssumeRole(statement) {
			continue
		}
		statementCond, _ := statement["Condition"].(map[string]interface{})
		statement["Condition"] = mergeSTSExternalIDIntoCondition(statementCond, condition)
	}
	updated, err := json.Marshal(builtDoc)
	if err != nil {
		return "", fmt.Errorf("failed to marshal rebuilt trust policy: %w", err)
	}
	return string(updated), nil
}

func stsExternalIDConditionFromTrustPolicy(policyJSON string) (map[string]interface{}, error) {
	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(policyJSON), &doc); err != nil {
		return nil, fmt.Errorf("failed to parse existing trust policy: %w", err)
	}
	statements, ok := doc["Statement"].([]interface{})
	if !ok {
		return nil, nil
	}
	for _, raw := range statements {
		statement, ok := raw.(map[string]interface{})
		if !ok || !trustPolicyStatementAllowsAssumeRole(statement) {
			continue
		}
		condition, ok := statement["Condition"].(map[string]interface{})
		if !ok || !trustPolicyConditionHasExternalID(condition) {
			continue
		}
		return condition, nil
	}
	return nil, nil
}

func trustPolicyStatementAllowsAssumeRole(statement map[string]interface{}) bool {
	if statement["Effect"] != policyEffectAllow {
		return false
	}
	switch action := statement["Action"].(type) {
	case string:
		return action == "sts:AssumeRole"
	case []interface{}:
		for _, raw := range action {
			if actionStr, ok := raw.(string); ok && actionStr == "sts:AssumeRole" {
				return true
			}
		}
	}
	return false
}

func mergeSTSExternalIDIntoCondition(
	destCondition, sourceCondition map[string]interface{},
) map[string]interface{} {
	if destCondition == nil {
		destCondition = map[string]interface{}{}
	}
	for _, operator := range []string{"StringEquals", "StringEqualsIfExists"} {
		sourceRaw, ok := sourceCondition[operator]
		if !ok {
			continue
		}
		sourceBlock, ok := sourceRaw.(map[string]interface{})
		if !ok {
			continue
		}
		externalIDValue, ok := sourceBlock["sts:ExternalId"]
		if !ok {
			continue
		}
		destRaw, ok := destCondition[operator]
		var destBlock map[string]interface{}
		if ok {
			destBlock, ok = destRaw.(map[string]interface{})
		}
		if !ok || destBlock == nil {
			destBlock = map[string]interface{}{}
		}
		destBlock["sts:ExternalId"] = externalIDValue
		destCondition[operator] = destBlock
	}
	return destCondition
}

func trustPolicyConditionHasExternalID(condition map[string]interface{}) bool {
	for _, operator := range []string{"StringEquals", "StringEqualsIfExists"} {
		raw, ok := condition[operator]
		if !ok {
			continue
		}
		operatorBlock, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if _, ok := operatorBlock["sts:ExternalId"]; ok {
			return true
		}
	}
	return false
}

// STSExternalIDClusterResolution is the outcome of validating or discovering a cluster STS external ID.
type STSExternalIDClusterResolution struct {
	ExternalID string
	// Ambiguous is true when role trust policies define ExternalId values but discovery cannot select one.
	Ambiguous bool
	// MismatchedTrustPolicies is true when installer and support both define ExternalIds with none in common.
	MismatchedTrustPolicies bool
}

// ResolveSTSExternalIDForClusterDetails validates or discovers the STS external ID from installer and
// support trust policies and reports whether discovery was ambiguous.
func ResolveSTSExternalIDForClusterDetails(
	entered, installerRoleARN, supportRoleARN string, client Client,
) (STSExternalIDClusterResolution, error) {
	installerPolicy, err := TrustPolicyJSONForRoleARN(client, installerRoleARN)
	if err != nil {
		return STSExternalIDClusterResolution{}, err
	}
	supportPolicy, err := TrustPolicyJSONForRoleARN(client, supportRoleARN)
	if err != nil {
		return STSExternalIDClusterResolution{}, err
	}
	if entered != "" {
		if err := ststrust.ValidateEnteredForRoleTrustPolicies(entered, installerPolicy, supportPolicy); err != nil {
			return STSExternalIDClusterResolution{}, err
		}
		return STSExternalIDClusterResolution{ExternalID: entered}, nil
	}
	externalID, err := ststrust.DiscoverSTSExternalID(installerPolicy, supportPolicy)
	if err != nil {
		return STSExternalIDClusterResolution{}, err
	}
	ambiguous := false
	mismatched := false
	if externalID == "" {
		installerIDs, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(installerPolicy)
		if err != nil {
			return STSExternalIDClusterResolution{}, err
		}
		supportIDs, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(supportPolicy)
		if err != nil {
			return STSExternalIDClusterResolution{}, err
		}
		ambiguous, err = isSTSExternalIDDiscoveryAmbiguous(installerPolicy, supportPolicy)
		if err != nil {
			return STSExternalIDClusterResolution{}, err
		}
		mismatched = hasMismatchedSTSExternalIDTrustPolicies(installerIDs, supportIDs)
	}
	return STSExternalIDClusterResolution{
		ExternalID:              externalID,
		Ambiguous:               ambiguous,
		MismatchedTrustPolicies: mismatched,
	}, nil
}

// ResolveSTSExternalIDForCluster validates or discovers the STS external ID from installer and support trust policies.
func ResolveSTSExternalIDForCluster(entered, installerRoleARN, supportRoleARN string, client Client) (string, error) {
	result, err := ResolveSTSExternalIDForClusterDetails(entered, installerRoleARN, supportRoleARN, client)
	if err != nil {
		return "", err
	}
	return result.ExternalID, nil
}

// isSTSExternalIDDiscoveryAmbiguous reports whether trust policies contain ExternalId values but discovery
// cannot select a single ID. Policies with no ExternalId are not ambiguous.
func isSTSExternalIDDiscoveryAmbiguous(installerPolicy, supportPolicy string) (bool, error) {
	installerIDs, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(installerPolicy)
	if err != nil {
		return false, err
	}
	supportIDs, err := ststrust.CollectSTSExternalIDsFromTrustPolicy(supportPolicy)
	if err != nil {
		return false, err
	}
	if len(installerIDs) == 0 && len(supportIDs) == 0 {
		return false, nil
	}
	discovered, err := ststrust.DiscoverSTSExternalID(installerPolicy, supportPolicy)
	if err != nil {
		return false, err
	}
	return discovered == "", nil
}

// hasMismatchedSTSExternalIDTrustPolicies reports when both roles define ExternalIds but share none.
func hasMismatchedSTSExternalIDTrustPolicies(installerIDs, supportIDs []string) bool {
	if len(installerIDs) == 0 || len(supportIDs) == 0 {
		return false
	}
	lookup := make(map[string]struct{}, len(installerIDs))
	for _, id := range installerIDs {
		lookup[id] = struct{}{}
	}
	for _, id := range supportIDs {
		if _, ok := lookup[id]; ok {
			return false
		}
	}
	return true
}

// ValidateSTSExternalIDFormat validates user input when --external-id is set.
func ValidateSTSExternalIDFormat(externalID string) error {
	if externalID == "" {
		return nil
	}
	return ststrust.ValidateSTSExternalIDFormat(externalID)
}
