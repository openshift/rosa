package output

import (
	"bytes"
	"fmt"
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	reportertest "github.com/openshift/rosa/test/reporter"
)

var _ = Describe("StructuredReporter", func() {

	var (
		origStderr *os.File
		readPipe   *os.File
		writePipe  *os.File
	)

	BeforeEach(func() {
		SetOutput("")
		origStderr = os.Stderr
		var err error
		readPipe, writePipe, err = os.Pipe()
		Expect(err).ToNot(HaveOccurred())
		os.Stderr = writePipe
	})

	AfterEach(func() {
		os.Stderr = origStderr
		SetOutput("")
		readPipe.Close()
	})

	captureStderr := func() string {
		writePipe.Close()
		os.Stderr = origStderr
		var buf bytes.Buffer
		_, err := io.Copy(&buf, readPipe)
		Expect(err).ToNot(HaveOccurred())
		return buf.String()
	}

	Context("Errorf", func() {
		It("delegates to inner reporter when no output flag is set", func() {
			errCalled := false
			fake := &reportertest.FakeLogger{ErrorFn: func(string, ...any) error { errCalled = true; return nil }}
			NewStructuredReporter(fake).Errorf("something went wrong")
			captured := captureStderr()
			Expect(errCalled).To(BeTrue())
			Expect(captured).To(BeEmpty())
		})

		It("formats args correctly before delegating when no flag is set", func() {
			var lastErr string
			fake := &reportertest.FakeLogger{ErrorFn: func(f string, a ...any) error { lastErr = fmt.Sprintf(f, a...); return nil }}
			NewStructuredReporter(fake).Errorf("failed: %s", "bad token")
			captureStderr()
			Expect(lastErr).To(Equal("failed: bad token"))
		})

		It("prints JSON to stderr and skips inner reporter when JSON flag is set", func() {
			SetOutput(JSON)
			errCalled := false
			fake := &reportertest.FakeLogger{ErrorFn: func(string, ...any) error { errCalled = true; return nil }}
			NewStructuredReporter(fake).Errorf("connection failed")
			captured := captureStderr()
			Expect(errCalled).To(BeFalse())
			Expect(captured).To(ContainSubstring(`"error"`))
			Expect(captured).To(ContainSubstring("connection failed"))
		})

		It("prints JSON to stderr and skips inner reporter when YAML flag is set", func() {
			SetOutput(YAML)
			errCalled := false
			fake := &reportertest.FakeLogger{ErrorFn: func(string, ...any) error { errCalled = true; return nil }}
			NewStructuredReporter(fake).Errorf("connection failed")
			captured := captureStderr()
			Expect(errCalled).To(BeFalse())
			Expect(captured).To(ContainSubstring(`"error"`))
			Expect(captured).To(ContainSubstring("connection failed"))
		})
	})

	Context("Warnf", func() {
		It("delegates to inner reporter when no output flag is set", func() {
			warnCalled := false
			fake := &reportertest.FakeLogger{WarnFn: func(string, ...any) { warnCalled = true }}
			NewStructuredReporter(fake).Warnf("region mismatch")
			captured := captureStderr()
			Expect(warnCalled).To(BeTrue())
			Expect(captured).To(BeEmpty())
		})

		It("prints JSON to stderr and skips inner reporter when JSON flag is set", func() {
			SetOutput(JSON)
			warnCalled := false
			fake := &reportertest.FakeLogger{WarnFn: func(string, ...any) { warnCalled = true }}
			NewStructuredReporter(fake).Warnf("region mismatch")
			captured := captureStderr()
			Expect(warnCalled).To(BeFalse())
			Expect(captured).To(ContainSubstring(`"warning"`))
			Expect(captured).To(ContainSubstring("region mismatch"))
		})

		It("prints JSON to stderr and skips inner reporter when YAML flag is set", func() {
			SetOutput(YAML)
			warnCalled := false
			fake := &reportertest.FakeLogger{WarnFn: func(string, ...any) { warnCalled = true }}
			NewStructuredReporter(fake).Warnf("region mismatch")
			captured := captureStderr()
			Expect(warnCalled).To(BeFalse())
			Expect(captured).To(ContainSubstring(`"warning"`))
			Expect(captured).To(ContainSubstring("region mismatch"))
		})
	})

	Context("passthrough methods", func() {
		It("Infof delegates to inner", func() {
			var lastInfo string
			fake := &reportertest.FakeLogger{InfoFn: func(f string, a ...any) { lastInfo = fmt.Sprintf(f, a...) }}
			NewStructuredReporter(fake).Infof("hello %s", "world")
			captureStderr()
			Expect(lastInfo).To(Equal("hello world"))
		})

		It("Debugf delegates to inner", func() {
			var lastDebug string
			fake := &reportertest.FakeLogger{DebugFn: func(f string, a ...any) { lastDebug = fmt.Sprintf(f, a...) }}
			NewStructuredReporter(fake).Debugf("debug %s", "msg")
			captureStderr()
			Expect(lastDebug).To(Equal("debug msg"))
		})

		It("IsTerminal delegates to inner", func() {
			Expect(NewStructuredReporter(&reportertest.FakeLogger{}).IsTerminal()).To(BeFalse())
		})
	})
})
