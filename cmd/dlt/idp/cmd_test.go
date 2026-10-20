// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package idp

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/utils/helper"
)

var _ = Describe("Delete IDP", func() {
	Context("Args validator", func() {
		It("rejects zero arguments", func() {
			err := Cmd.Args(nil, []string{})
			helper.ExpectErrorWithMessage(err, "expected exactly one")
		})

		It("accepts exactly one argument", func() {
			err := Cmd.Args(nil, []string{"github-1"})
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects two arguments", func() {
			err := Cmd.Args(nil, []string{"github-1", "extra"})
			helper.ExpectErrorWithMessage(err, "expected exactly one")
		})
	})
})
