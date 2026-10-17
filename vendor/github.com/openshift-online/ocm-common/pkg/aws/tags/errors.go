package tags

import "errors"

// Sentinel errors for AWS tag map validation and diff helpers.
var (
	// ErrEmptyKey is returned when a tag key is empty.
	ErrEmptyKey = errors.New("AWS tag key cannot be empty")
	// ErrKeyTooLong is returned when a tag key exceeds MaxKeyLength.
	ErrKeyTooLong = errors.New("AWS tag key exceeds maximum length")
	// ErrValueTooLong is returned when a tag value exceeds MaxValueLength.
	ErrValueTooLong = errors.New("AWS tag value exceeds maximum length")
	// ErrKeyFormat is returned when a tag key fails charset validation.
	ErrKeyFormat = errors.New("AWS tag key format is invalid")
	// ErrValueFormat is returned when a tag value fails charset validation.
	ErrValueFormat = errors.New("AWS tag value format is invalid")
	// ErrReservedKey is returned when a tag key is reserved by AWS or ROSA/OCM.
	ErrReservedKey = errors.New("AWS tag key is reserved")
)
