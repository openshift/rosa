// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package rosa

import (
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/debug"
)

var _ = Describe("NewRuntime", func() {
	var previousDebug bool

	BeforeEach(func() {
		previousDebug = debug.Enabled()
	})

	AfterEach(func() {
		debug.SetEnabled(previousDebug)
	})

	captureStderr := func(fn func()) string {
		original := os.Stderr
		r, w, err := os.Pipe()
		Expect(err).NotTo(HaveOccurred())
		defer func() { os.Stderr = original }()
		os.Stderr = w

		fn()

		Expect(w.Close()).To(Succeed())
		output, err := io.ReadAll(r)
		Expect(err).NotTo(HaveOccurred())
		return string(output)
	}

	It("writes nothing to the structured logger's stream by default", func() {
		debug.SetEnabled(false)
		output := captureStderr(func() {
			r := NewRuntime()
			r.StructuredLogger.Info("should not appear")
		})
		Expect(output).To(BeEmpty())
	})

	It("writes to stderr once --debug is enabled", func() {
		debug.SetEnabled(true)
		output := captureStderr(func() {
			r := NewRuntime()
			r.StructuredLogger.Info("should appear")
		})
		Expect(output).To(ContainSubstring("should appear"))
	})
})
