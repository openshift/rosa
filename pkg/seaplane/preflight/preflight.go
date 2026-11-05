// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package preflight

import (
	"context"

	"github.com/openshift/rosa/pkg/seaplane/pathbind"
)

// PreflightState is the outcome of a single preflight check.
type PreflightState string

const (
	// PreflightPass means the check succeeded.
	PreflightPass PreflightState = "pass"
	// PreflightWarn means the check found a condition worth reporting to the
	// user but that does not block the create request.
	PreflightWarn PreflightState = "warn"
	// PreflightFail means the check failed and the create request must not be sent.
	PreflightFail PreflightState = "fail"
)

// PreflightResult is the structured outcome of one preflight check. Consumers
// decide how to surface results (terminal output, TF diagnostics, controller
// conditions); checks themselves must not print.
type PreflightResult struct {
	// Name identifies the check.
	Name string
	// State is the outcome of the check.
	State PreflightState
	// Message describes the outcome; it is the error text for failures.
	Message string
	// DocsURL optionally points at documentation for remediating a failure.
	DocsURL string
}

// PreflightCheck validates and prepares one aspect of a cluster create request
// before it is sent to the Platform API.
//
// Checks run in registration order and may mutate input (bridging CLI-provided
// values into the request shape). Anything a check needs beyond the input —
// the runtime, AWS clients, reporters — is captured when the check is
// constructed, so the interface itself stays free of CLI and runtime
// dependencies and the check bodies can later be extracted into a module
// shared with the Terraform provider and CAPA.
type PreflightCheck interface {
	// Name identifies the check in results and error aggregation.
	Name() string
	// Run executes the check. A non-nil error fails the preflight run
	// immediately; its message is preserved verbatim.
	Run(ctx context.Context, input *pathbind.ClusterCreateInput) error
}

// preflightCheck adapts a named function to the PreflightCheck interface.
type preflightCheck struct {
	name string
	fn   func(ctx context.Context, input *pathbind.ClusterCreateInput) error
}

// NewPreflightCheck wraps a function as a PreflightCheck.
func NewPreflightCheck(
	name string,
	fn func(ctx context.Context, input *pathbind.ClusterCreateInput) error,
) PreflightCheck {
	return &preflightCheck{name: name, fn: fn}
}

func (c *preflightCheck) Name() string { return c.name }

func (c *preflightCheck) Run(ctx context.Context, input *pathbind.ClusterCreateInput) error {
	return c.fn(ctx, input)
}

// RunPreflightChecks executes checks in order, failing fast on the first
// error. Results for every executed check are returned alongside the error so
// callers can report progress; when err is non-nil the last result is the
// failure that aborted the run and no further checks were executed.
func RunPreflightChecks(
	ctx context.Context,
	checks []PreflightCheck,
	input *pathbind.ClusterCreateInput,
) ([]PreflightResult, error) {
	results := make([]PreflightResult, 0, len(checks))
	for _, check := range checks {
		err := check.Run(ctx, input)
		if err != nil {
			results = append(results, PreflightResult{
				Name:    check.Name(),
				State:   PreflightFail,
				Message: err.Error(),
			})
			// Return the error unwrapped so callers surface the check's own
			// (v1-compatible) message verbatim.
			return results, err
		}
		results = append(results, PreflightResult{
			Name:  check.Name(),
			State: PreflightPass,
		})
	}
	return results, nil
}
