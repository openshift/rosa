/*
Copyright (c) 2026 Red Hat, Inc.

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

package log

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Log Suite")
}

var _ = Describe("Log", func() {
	Describe("FromContext", func() {
		It("returns a discard logger by default", func() {
			logger := FromContext(context.Background())
			Expect(logger).NotTo(BeNil())
			// A discard logger must not panic and must not be observable via output.
			logger.Info("this should go nowhere")
		})

		It("returns a discard logger for a nil logger", func() {
			var nilLogger *slog.Logger
			ctx := WithLogger(context.Background(), nilLogger)

			logger := FromContext(ctx)
			Expect(logger).NotTo(BeNil())
			// Must not panic even though a nil *slog.Logger was attached.
			logger.Info("this should go nowhere")
		})
	})

	Describe("WithLogger", func() {
		It("round-trips the attached logger", func() {
			var buf bytes.Buffer
			logger := New(&buf, slog.LevelInfo)

			ctx := WithLogger(context.Background(), logger)
			got := FromContext(ctx)

			got.Info("hello", "key", "value")

			Expect(buf.Len()).NotTo(BeZero())
		})
	})

	Describe("New", func() {
		It("respects the configured level", func() {
			var buf bytes.Buffer
			logger := New(&buf, slog.LevelInfo)

			logger.Debug("should be filtered out")
			Expect(buf.Len()).To(BeZero())

			logger.Info("should be emitted")
			Expect(buf.Len()).NotTo(BeZero())
		})
	})
})
