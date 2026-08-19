package oc

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	reportertest "github.com/openshift/rosa/test/reporter"
)

var _ = Describe("verify oc", func() {
	It("Warns when oc is not installed", func() {
		var lastWarn string
		r := &reportertest.FakeLogger{
			Terminal: true,
			WarnFn:   func(f string, a ...any) { lastWarn = fmt.Sprintf(f, a...) },
		}
		getVersion := func(context.Context) ([]byte, error) {
			return nil, fmt.Errorf("exec: \"oc\": executable file not found in $PATH")
		}

		runVerifyOC(context.Background(), r, getVersion)

		Expect(lastWarn).To(ContainSubstring("not installed"), "expected missing-binary warning text")
	})

	It("Reports correct version when 4.x is installed", func() {
		warnCalled, errCalled := false, false
		infoCount := 0
		var lastInfo string
		r := &reportertest.FakeLogger{
			Terminal: true,
			WarnFn:   func(string, ...any) { warnCalled = true },
			ErrorFn:  func(string, ...any) error { errCalled = true; return nil },
			InfoFn:   func(f string, a ...any) { infoCount++; lastInfo = fmt.Sprintf(f, a...) },
		}
		getVersion := func(context.Context) ([]byte, error) {
			return []byte("Client Version: 4.14.0\n"), nil
		}

		runVerifyOC(context.Background(), r, getVersion)

		Expect(warnCalled).To(BeFalse(), "expected no warnings for supported 4.x version")
		Expect(errCalled).To(BeFalse(), "expected no errors for supported 4.x version")
		Expect(infoCount).To(Equal(2), "expected start and success info messages in terminal mode")
		Expect(lastInfo).To(ContainSubstring("4.14.0"), "expected success info to mention the detected version")
	})

	It("Warns when oc version is not 4.x", func() {
		warnCount := 0
		var firstWarn, lastWarn string
		r := &reportertest.FakeLogger{
			Terminal: true,
			WarnFn: func(f string, a ...any) {
				warnCount++
				msg := fmt.Sprintf(f, a...)
				if warnCount == 1 {
					firstWarn = msg
				}
				lastWarn = msg
			},
		}
		getVersion := func(context.Context) ([]byte, error) {
			return []byte("Client Version: 3.11.0\n"), nil
		}

		runVerifyOC(context.Background(), r, getVersion)

		Expect(warnCount).To(Equal(2), "expected version and unsupported warnings for non-4.x oc")
		Expect(firstWarn).To(ContainSubstring("3.11.0"), "expected first warning to include the detected version")
		Expect(lastWarn).To(ContainSubstring("not supported"), "expected unsupported-version warning text")
	})

	It("Reports an error when oc returns output and an execution error", func() {
		var lastErr string
		r := &reportertest.FakeLogger{
			Terminal: true,
			ErrorFn:  func(f string, a ...any) error { lastErr = fmt.Sprintf(f, a...); return nil },
		}
		getVersion := func(context.Context) ([]byte, error) {
			return []byte("Client Version: 4.14.0\n"), fmt.Errorf("permission denied")
		}

		runVerifyOC(context.Background(), r, getVersion)

		Expect(lastErr).To(ContainSubstring("permission denied"), "expected the execution error to be surfaced")
	})

	It("Does not print verifying message when not in terminal mode", func() {
		infoCalled := false
		r := &reportertest.FakeLogger{
			Terminal: false,
			InfoFn:   func(string, ...any) { infoCalled = true },
		}
		getVersion := func(context.Context) ([]byte, error) {
			return []byte("Client Version: 4.14.0\n"), nil
		}

		runVerifyOC(context.Background(), r, getVersion)

		Expect(infoCalled).To(BeFalse(), "expected no informational output in non-terminal mode")
	})

	It("Rejects extra arguments via cobra.NoArgs", func() {
		Expect(Cmd.Args).NotTo(BeNil(), "expected cobra.NoArgs validation to be configured")
		err := Cmd.Args(Cmd, []string{"unexpected"})
		Expect(err).To(HaveOccurred(), "expected extra arguments to be rejected")
	})
})
