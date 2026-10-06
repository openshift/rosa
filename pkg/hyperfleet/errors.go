package hyperfleet

import (
	"errors"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// WithAPIErrorDetails includes field-level validation causes returned by the Platform API.
func WithAPIErrorDetails(err error) error {
	var statusError apierrors.APIStatus
	if !errors.As(err, &statusError) {
		return err
	}
	details := statusError.Status().Details
	if details == nil || len(details.Causes) == 0 {
		return err
	}
	causes := make([]string, 0, len(details.Causes))
	for _, cause := range details.Causes {
		if cause.Field != "" {
			causes = append(causes, fmt.Sprintf("%s: %s", cause.Field, cause.Message))
		} else {
			causes = append(causes, cause.Message)
		}
	}
	return fmt.Errorf("%w (%s)", err, strings.Join(causes, "; "))
}
