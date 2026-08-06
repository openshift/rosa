/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package errors holds shared error types for the CLI/core boundary
// described in guidelines/error-conventions.md. It has no dependencies of
// its own, so any core workflow package can import it without pulling in
// CLI or presentation concerns.
//
// Because it shares its name with the standard library "errors" package,
// callers that also use errors.Is/errors.As import this package under an
// alias, e.g. rosaerrors "github.com/openshift/rosa/pkg/errors".
package errors

import "fmt"

// ValidationError marks a failure that occurred during domain validation.
// It serves two roles that share the same type so callers only ever need
// one import and one errors.As target:
//
//   - A leaf-level error from a single check, e.g.
//     &ValidationError{Field: "ClusterName", Message: "cluster name is required"}.
//     Field is the struct field the check applies to, when the check maps
//     to exactly one field; it is metadata for structured consumers (a
//     future --output json mode, a REST API, a TUI) and is never rendered
//     by Error() itself, so introducing Field never changes existing
//     user-facing text.
//   - The aggregate wrapper a workflow function places around its
//     Request's Validate() call, e.g.
//     &ValidationError{Err: fmt.Errorf("invalid request: %w", err)}.
//     This guarantees every Validate() failure is classified as a
//     validation error regardless of what individual checks return,
//     without relying on every check remembering to use this type.
//
// Callers use errors.As to distinguish invalid input (no side effects
// occurred) from operational failures encountered later in a workflow,
// without parsing error text. This type is shared by every
// Request/Validate() workflow; workflows do not define their own
// <Verb><Resource>ValidationError. See "Classifying validation errors" in
// guidelines/workflow-conventions.md.
type ValidationError struct {
	// Field is the request field that failed validation, when the failure
	// maps to exactly one field. Empty for checks that span more than one
	// field, and for the aggregate wrapper case.
	Field string

	// Message describes the violation. Empty for the aggregate wrapper
	// case, where Error() delegates to Err instead.
	Message string

	// Err is the underlying cause, when there is one: either a wrapped
	// error from a lower-level check, or the aggregate error a workflow
	// function wraps at its Validate() call site. Preserved for
	// errors.Is/errors.As via Unwrap().
	Err error
}

func (e *ValidationError) Error() string {
	switch {
	case e.Message != "" && e.Err != nil:
		return fmt.Sprintf("%s: %s", e.Message, e.Err)
	case e.Message != "":
		return e.Message
	case e.Err != nil:
		return e.Err.Error()
	case e.Field != "":
		return fmt.Sprintf("validation failed: %s", e.Field)
	default:
		return "validation failed"
	}
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}
