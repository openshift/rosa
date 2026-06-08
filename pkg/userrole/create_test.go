/*
Copyright (c) 2021 Red Hat, Inc.

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

package userrole_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/userrole"
)

var _ = Describe("User role create helpers", func() {
	It("validates prefix and optional fields", func() {
		err := userrole.Validate(userrole.Input{
			Prefix: "ManagedOpenShift",
			Mode:   interactive.ModeAuto,
		})
		Expect(err).NotTo(HaveOccurred())

		err = userrole.Validate(userrole.Input{
			Prefix: "bad prefix",
			Mode:   interactive.ModeAuto,
		})
		Expect(err).To(HaveOccurred())
	})

	It("builds manual mode commands", func() {
		creator := &aws.Creator{
			AccountID: "123456789012",
			Partition: "aws",
		}
		commands := userrole.BuildCommands(
			"ManagedOpenShift",
			"",
			"jdoe",
			creator,
			"production",
			"",
		)
		Expect(commands).To(ContainSubstring("rosa link user-role --role-arn"))
		Expect(commands).To(ContainSubstring("123456789012"))
	})
})
