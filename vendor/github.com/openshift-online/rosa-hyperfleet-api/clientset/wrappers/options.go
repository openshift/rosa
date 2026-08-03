/*
Copyright 2026.

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

package wrappers

// GetOptions configures a single-resource read.
// Currently only the default behavior is supported.
type GetOptions struct{}

// CreateOptions configures a Create request.
// Dry-run, field manager, and field validation are currently unsupported.
type CreateOptions struct{}

// UpdateOptions configures an Update request.
// Dry-run, field manager, and field validation are currently unsupported.
type UpdateOptions struct{}

// PatchOptions configures a Patch request.
// Dry-run, force, and field manager are currently unsupported.
type PatchOptions struct{}

// DeleteOptions configures a Delete request.
// Grace period, propagation policy, and preconditions are currently unsupported.
type DeleteOptions struct{}

// ListOptions configures a List request.
type ListOptions struct {
	// Limit caps the number of items returned. Maximum 100, default 50.
	Limit int64
	// Offset is the number of items to skip before returning results.
	Offset int64
}
