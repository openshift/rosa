package ststrust

import (
	"fmt"
)

// Role labels used in validation and mismatch errors.
const (
	roleLabelInstaller = "installer role"
	roleLabelSupport   = "support role"
)

// ValidateSTSExternalIDFormat validates length and character set for aws.sts.external_id.
func ValidateSTSExternalIDFormat(entered string) error {
	if entered == "" {
		return ErrExternalIDEmpty
	}
	if len(entered) < MinSTSExternalIDLength {
		return fmt.Errorf("%w: must be at least %d characters", ErrExternalIDFormat, MinSTSExternalIDLength)
	}
	if len(entered) > MaxSTSExternalIDLength {
		return fmt.Errorf("%w: must be at most %d characters", ErrExternalIDFormat, MaxSTSExternalIDLength)
	}
	if !STSExternalIDRegex.MatchString(entered) {
		return fmt.Errorf("%w: must match %s", ErrExternalIDFormat, STSExternalIDRegex.String())
	}
	return nil
}

// ValidateEnteredForRoleTrustPolicies ensures entered is present in each non-empty role trust policy.
// Empty installer or support policy strings are skipped. Callers must supply a user-entered external ID.
func ValidateEnteredForRoleTrustPolicies(entered, installerPolicy, supportPolicy string) error {
	if err := ValidateSTSExternalIDFormat(entered); err != nil {
		return err
	}
	if installerPolicy == "" && supportPolicy == "" {
		return fmt.Errorf("%w: no installer or support trust policy provided", ErrNoTrustPolicyExternalID)
	}
	if installerPolicy != "" {
		if err := validateEnteredForPolicy(entered, installerPolicy, roleLabelInstaller); err != nil {
			return err
		}
	}
	if supportPolicy != "" {
		if err := validateEnteredForPolicy(entered, supportPolicy, roleLabelSupport); err != nil {
			return err
		}
	}
	return nil
}

// validateEnteredForPolicy checks that entered appears in a single role trust policy.
func validateEnteredForPolicy(entered, policyJSON, roleLabel string) error {
	ids, err := CollectSTSExternalIDsFromTrustPolicy(policyJSON)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return fmt.Errorf("%w for %s", ErrNoTrustPolicyExternalID, roleLabel)
	}
	for _, id := range ids {
		if id == entered {
			return nil
		}
	}
	return &ExternalIDMismatchError{
		RoleLabel:   roleLabel,
		Entered:     entered,
		FoundInRole: ids,
	}
}
