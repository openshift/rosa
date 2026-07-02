/*
Copyright (c) 2025 Red Hat, Inc.

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

package output

import (
	"fmt"

	"github.com/openshift/rosa/pkg/reporter"
)

// StructuredReporter wraps a reporter.Logger so that Errorf and Warnf
// emit JSON to stderr when a structured output flag (--output json/yaml)
// is set, suppressing the plain-text prefix in that case.
type StructuredReporter struct {
	inner reporter.Logger
}

// NewStructuredReporter returns a reporter.Logger that formats errors and
// warnings as JSON when the --output flag is active, and otherwise
// delegates to the provided reporter unchanged.
func NewStructuredReporter(r reporter.Logger) reporter.Logger {
	return &StructuredReporter{inner: r}
}

func (r *StructuredReporter) Errorf(format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	if !PrintError(err) {
		return r.inner.Errorf(format, args...)
	}
	return err
}

func (r *StructuredReporter) Warnf(format string, args ...any) {
	if !PrintWarn(fmt.Errorf(format, args...)) {
		r.inner.Warnf(format, args...)
	}
}

func (r *StructuredReporter) Debugf(format string, args ...any) {
	r.inner.Debugf(format, args...)
}

func (r *StructuredReporter) Infof(format string, args ...any) {
	r.inner.Infof(format, args...)
}

func (r *StructuredReporter) IsTerminal() bool {
	return r.inner.IsTerminal()
}
