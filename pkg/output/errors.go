package output

import (
	sdkErr "github.com/openshift-online/ocm-sdk-go/errors"
	"github.com/pkg/errors"
)

// Converts OCM errors to user-friendly strings
func ErrorToString(err error) string {
	switch err := errors.Cause(err).(type) {
	case *sdkErr.Error:
		return err.Reason()
	default:
		return "Unknown error"
	}
}
