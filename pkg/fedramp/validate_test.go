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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	rosaerrors "github.com/openshift/rosa/pkg/errors"
)

var _ = Describe("ValidateGovCloudFields", func() {
	It("returns no errors when isGovcloud is false, regardless of the other fields", func() {
		Expect(ValidateGovCloudFields(false, "", "")).To(BeEmpty())
	})

	It("returns no errors when isGovcloud is true and both fields are present", func() {
		Expect(ValidateGovCloudFields(true, "111222333444", "aws-us-gov")).To(BeEmpty())
	})

	It("reports a missing account ID", func() {
		errs := ValidateGovCloudFields(true, "", "aws-us-gov")
		Expect(errs).To(HaveLen(1))
		var validationErr *rosaerrors.ValidationError
		Expect(errors.As(errs[0], &validationErr)).To(BeTrue())
		Expect(validationErr.Field).To(Equal("AccountID"))
	})

	It("reports a missing partition", func() {
		errs := ValidateGovCloudFields(true, "111222333444", "")
		Expect(errs).To(HaveLen(1))
		var validationErr *rosaerrors.ValidationError
		Expect(errors.As(errs[0], &validationErr)).To(BeTrue())
		Expect(validationErr.Field).To(Equal("Partition"))
	})

	It("reports both violations when account ID and partition are empty", func() {
		Expect(ValidateGovCloudFields(true, "", "")).To(HaveLen(2))
	})
})
