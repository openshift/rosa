// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package rosacli

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ReflectTuningConfigDescription", func() {
	It("parses a description whose Spec block scalar has deeply indented, dash-less JSON content", func() {
		output := `
Name:                       my-tuning
ID:                         abc123
Spec:                       {
                              "profile": [
                                {
                                  "data": "[sysctl]\nvm.dirty_ratio=55\n",
                                  "name": "tuned-1-profile"
                                }
                              ]
                            }
`
		tcs := &tuningConfigService{ResourcesService: ResourcesService{client: &Client{Parser: NewParser()}}}
		var buf bytes.Buffer
		buf.WriteString(output)

		description, err := tcs.ReflectTuningConfigDescription(buf)

		Expect(err).ToNot(HaveOccurred())
		Expect(description.Name).To(Equal("my-tuning"))
		Expect(description.Spec).To(ContainSubstring("tuned-1-profile"))
		Expect(description.Spec).To(ContainSubstring("vm.dirty_ratio"))
	})
})
