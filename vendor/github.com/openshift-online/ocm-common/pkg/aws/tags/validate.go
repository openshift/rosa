package tags

import (
	"fmt"
	"strings"
)

// Validate checks an AWS custom tag map for write-side rules: non-empty keys,
// key/value length and charset, and reserved keys/prefixes.
// Tag count limits are enforced by the OCM API, not this package.
// A nil map is treated as empty and is valid.
func Validate(tags map[string]string) error {
	for key, value := range tags {
		if err := ValidateKey(key); err != nil {
			return err
		}
		if err := ValidateValue(value); err != nil {
			return fmt.Errorf("%w for key %q", err, key)
		}
	}
	return nil
}

// ValidateKey validates a single AWS tag key.
func ValidateKey(key string) error {
	if key == "" {
		return ErrEmptyKey
	}
	if len(key) > MaxKeyLength {
		return fmt.Errorf("%w: %d > %d", ErrKeyTooLong, len(key), MaxKeyLength)
	}
	if !KeyRE.MatchString(key) {
		return fmt.Errorf("%w: must match %s", ErrKeyFormat, KeyRE.String())
	}
	if isReservedKey(key) {
		return fmt.Errorf("%w: %q", ErrReservedKey, key)
	}
	return nil
}

// ValidateValue validates a single AWS tag value (empty values are allowed).
func ValidateValue(value string) error {
	if len(value) > MaxValueLength {
		return fmt.Errorf("%w: %d > %d", ErrValueTooLong, len(value), MaxValueLength)
	}
	if !ValueRE.MatchString(value) {
		return fmt.Errorf("%w: must match %s", ErrValueFormat, ValueRE.String())
	}
	return nil
}

func isReservedKey(key string) bool {
	if _, ok := reservedExactKeys[key]; ok {
		return true
	}
	if strings.HasPrefix(strings.ToLower(key), ReservedKeyPrefixAWS) {
		return true
	}
	if strings.HasPrefix(key, ReservedKeyPrefixKubernetesCluster) {
		return true
	}
	return false
}
