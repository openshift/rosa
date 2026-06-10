// Package ststrust provides helpers for STS external ID validation and IAM trust policy handling.
//
// External IDs are never auto-generated; callers supply user-provided values. Trust policy JSON may
// be percent-encoded (see decodePolicyDocument); PathUnescape is used so '+' in external IDs is preserved.
package ststrust
