package utils

import "fmt"

const (
	OcpMajorVersion4 = 4
	OcpMajorVersion5 = 5
	V5MinorOffset    = 23 // OCP 5.0 == 4.23
)

// NormalizeToV4 converts an OCP version to the 4.x scheme for skew arithmetic.
// OCP 5.0 maps to 4.23, 5.1 to 4.24, etc. OCP 4.x versions pass through unchanged.
func NormalizeToV4(major, minor int) (int, int, error) {
	switch major {
	case OcpMajorVersion4:
		return major, minor, nil
	case OcpMajorVersion5:
		return OcpMajorVersion4, minor + V5MinorOffset, nil
	default:
		return 0, 0, fmt.Errorf("unsupported OCP major version: %d", major)
	}
}

// DenormalizeFromV4 converts a normalized 4.x minor version back to display form.
// A minor >= 23 maps to OCP 5.x (e.g., 23 → 5.0, 24 → 5.1). Otherwise returns 4.x.
func DenormalizeFromV4(minor int) (int, int) {
	if minor >= V5MinorOffset {
		return OcpMajorVersion5, minor - V5MinorOffset
	}
	return OcpMajorVersion4, minor
}
