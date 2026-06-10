package ststrust

import "errors"

// Sentinel errors for STS external ID validation and trust policy operations.
var (
	// ErrExternalIDEmpty is returned when an operation requires a non-empty external ID.
	ErrExternalIDEmpty = errors.New("STS external ID cannot be empty")
	// ErrExternalIDFormat is returned when the external ID fails format or length validation.
	ErrExternalIDFormat = errors.New("STS external ID format is invalid")
	// ErrExternalIDNotInTrustPolicy is returned when the entered ID is not present in a role trust policy.
	ErrExternalIDNotInTrustPolicy = errors.New("STS external ID is not present in role trust policy")
	// ErrExternalIDConflictOnInject is returned when injection would add an ID not already allowed by the trust policy.
	ErrExternalIDConflictOnInject = errors.New("STS external ID is not compatible with existing trust policy conditions")
	// ErrNoTrustPolicyExternalID is returned when validation requires ExternalId conditions but none were found.
	ErrNoTrustPolicyExternalID = errors.New("role trust policy has no sts:ExternalId condition")
)

// ExternalIDMismatchError describes a membership validation failure for a specific role.
type ExternalIDMismatchError struct {
	// RoleLabel identifies the installer or support role that failed validation.
	RoleLabel string
	// Entered is the user-supplied external ID.
	Entered string
	// FoundInRole lists sts:ExternalId values present in that role's trust policy.
	FoundInRole []string
}

// Error implements the error interface.
func (e *ExternalIDMismatchError) Error() string {
	return "STS external ID '" + e.Entered + "' does not match trust policy for " + e.RoleLabel +
		" (found: " + formatIDList(e.FoundInRole) + ")"
}

// Is reports whether target is ErrExternalIDNotInTrustPolicy.
func (e *ExternalIDMismatchError) Is(target error) bool {
	return target == ErrExternalIDNotInTrustPolicy
}
