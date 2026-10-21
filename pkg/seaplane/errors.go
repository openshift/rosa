package seaplane

import (
	"bytes"
	"encoding/json"
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

// WithAPIErrorBodyDetails adds the Platform API's non-standard errors field to
// the Kubernetes Status error returned by client-go.
func WithAPIErrorBodyDetails(err error, body []byte) error {
	detailedErr := WithAPIErrorDetails(err)
	if len(body) == 0 {
		return detailedErr
	}
	var response struct {
		Errors  json.RawMessage `json:"errors"`
		Details struct {
			Errors json.RawMessage `json:"errors"`
		} `json:"details"`
	}
	if json.Unmarshal(body, &response) != nil {
		return detailedErr
	}
	errorsJSON := response.Errors
	if isEmptyJSON(errorsJSON) {
		errorsJSON = response.Details.Errors
	}
	if isEmptyJSON(errorsJSON) {
		return detailedErr
	}
	if detailedErr == nil {
		return fmt.Errorf("platform API validation errors: %s", errorsJSON)
	}
	return fmt.Errorf("%w (errors: %s)", detailedErr, errorsJSON)
}

func isEmptyJSON(value json.RawMessage) bool {
	value = bytes.TrimSpace(value)
	return len(value) == 0 || bytes.Equal(value, []byte("null")) ||
		bytes.Equal(value, []byte("[]")) || bytes.Equal(value, []byte("{}"))
}
