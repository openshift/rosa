// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package output

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/reporter"
)

// fakeReporter records formatted messages for assertion in tests.
type fakeReporter struct {
	errors   []string
	warnings []string
	infos    []string
	debugs   []string
}

func (f *fakeReporter) Errorf(format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	f.errors = append(f.errors, msg)
	return errors.New(msg)
}

func (f *fakeReporter) Warnf(format string, args ...interface{}) {
	f.warnings = append(f.warnings, fmt.Sprintf(format, args...))
}

func (f *fakeReporter) Infof(format string, args ...interface{}) {
	f.infos = append(f.infos, fmt.Sprintf(format, args...))
}

func (f *fakeReporter) Debugf(format string, args ...interface{}) {
	f.debugs = append(f.debugs, fmt.Sprintf(format, args...))
}

func (f *fakeReporter) IsTerminal() bool { return false }

var _ reporter.Logger = &fakeReporter{}

var _ = Describe("StructuredReporter", func() {

	var (
		fake       *fakeReporter
		structured reporter.Logger
		origStderr *os.File
		readPipe   *os.File
		writePipe  *os.File
	)

	BeforeEach(func() {
		SetOutput("")
		fake = &fakeReporter{}
		structured = NewStructuredReporter(fake)
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
			structured.Errorf("something went wrong")
			captured := captureStderr()
			Expect(fake.errors).To(HaveLen(1))
			Expect(captured).To(BeEmpty())
		})

		It("formats args correctly before delegating when no flag is set", func() {
			structured.Errorf("failed: %s", "bad token")
			captureStderr()
			Expect(fake.errors).To(ContainElement("failed: bad token"))
		})

		It("prints JSON to stderr and skips inner reporter when JSON flag is set", func() {
			SetOutput(JSON)
			structured.Errorf("connection failed")
			captured := captureStderr()
			Expect(fake.errors).To(BeEmpty())
			Expect(captured).To(ContainSubstring(`"error"`))
			Expect(captured).To(ContainSubstring("connection failed"))
		})

		It("prints JSON to stderr and skips inner reporter when YAML flag is set", func() {
			SetOutput(YAML)
			structured.Errorf("connection failed")
			captured := captureStderr()
			Expect(fake.errors).To(BeEmpty())
			Expect(captured).To(ContainSubstring(`"error"`))
			Expect(captured).To(ContainSubstring("connection failed"))
		})
	})

	Context("Warnf", func() {
		It("delegates to inner reporter when no output flag is set", func() {
			structured.Warnf("region mismatch")
			captured := captureStderr()
			Expect(fake.warnings).To(HaveLen(1))
			Expect(captured).To(BeEmpty())
		})

		It("prints JSON to stderr and skips inner reporter when JSON flag is set", func() {
			SetOutput(JSON)
			structured.Warnf("region mismatch")
			captured := captureStderr()
			Expect(fake.warnings).To(BeEmpty())
			Expect(captured).To(ContainSubstring(`"warning"`))
			Expect(captured).To(ContainSubstring("region mismatch"))
		})

		It("prints JSON to stderr and skips inner reporter when YAML flag is set", func() {
			SetOutput(YAML)
			structured.Warnf("region mismatch")
			captured := captureStderr()
			Expect(fake.warnings).To(BeEmpty())
			Expect(captured).To(ContainSubstring(`"warning"`))
			Expect(captured).To(ContainSubstring("region mismatch"))
		})
	})

	Context("passthrough methods", func() {
		It("Infof delegates to inner", func() {
			structured.Infof("hello %s", "world")
			captureStderr()
			Expect(fake.infos).To(ContainElement("hello world"))
		})

		It("Debugf delegates to inner", func() {
			structured.Debugf("debug %s", "msg")
			captureStderr()
			Expect(fake.debugs).To(ContainElement("debug msg"))
		})

		It("IsTerminal delegates to inner", func() {
			Expect(structured.IsTerminal()).To(BeFalse())
		})
	})
})
