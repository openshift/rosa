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

package fedramp

import (
	rosaerrors "github.com/openshift/rosa/pkg/errors"
)

// ValidateGovCloudFields checks that accountID and partition are present
// when isGovcloud is true. It takes plain values, resolved by a caller from
// aws.Creator (via GetCreator) or an equivalent source, rather than a
// workflow-specific Request type, so any workflow that operates on a
// GovCloud-capable AWS account can validate the same invariant.
func ValidateGovCloudFields(isGovcloud bool, accountID, partition string) []error {
	if !isGovcloud {
		return nil
	}
	var errs []error
	if accountID == "" {
		errs = append(errs, &rosaerrors.ValidationError{
			Field: "AccountID", Message: "account ID is required for GovCloud environments",
		})
	}
	if partition == "" {
		errs = append(errs, &rosaerrors.ValidationError{
			Field: "Partition", Message: "partition is required for GovCloud environments",
		})
	}
	return errs
}
