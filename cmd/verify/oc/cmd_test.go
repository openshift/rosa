package oc

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type mockReporter struct {
	infos    []string
	warns    []string
	errs     []string
	terminal bool
}

func (m *mockReporter) Debugf(_ string, _ ...any) {}
func (m *mockReporter) Infof(f string, a ...any)  { m.infos = append(m.infos, fmt.Sprintf(f, a...)) }
func (m *mockReporter) Warnf(f string, a ...any)  { m.warns = append(m.warns, fmt.Sprintf(f, a...)) }
func (m *mockReporter) Errorf(f string, a ...any) error {
	m.errs = append(m.errs, fmt.Sprintf(f, a...))
	return nil
}
func (m *mockReporter) IsTerminal() bool { return m.terminal }

var _ = Describe("verify oc", func() {
	It("Warns when oc is not installed", func() {
		reporter := &mockReporter{terminal: true}
		getVersion := func(context.Context) ([]byte, error) {
			return nil, fmt.Errorf("exec: \"oc\": executable file not found in $PATH")
		}

		runVerifyOC(context.Background(), reporter, getVersion)

		Expect(reporter.warns).To(HaveLen(1), "expected one warning for missing oc binary")
		Expect(reporter.warns[0]).To(ContainSubstring("not installed"), "expected missing-binary warning text")
	})

	It("Reports correct version when 4.x is installed", func() {
		reporter := &mockReporter{terminal: true}
		getVersion := func(context.Context) ([]byte, error) {
			return []byte("Client Version: 4.14.0\n"), nil
		}

		runVerifyOC(context.Background(), reporter, getVersion)

		Expect(reporter.warns).To(BeEmpty(), "expected no warnings for supported 4.x version")
		Expect(reporter.errs).To(BeEmpty(), "expected no errors for supported 4.x version")
		Expect(reporter.infos).To(HaveLen(2), "expected start and success info messages in terminal mode")
		Expect(reporter.infos[1]).To(ContainSubstring("4.14.0"), "expected success info to mention the detected version")
	})

	It("Warns when oc version is not 4.x", func() {
		reporter := &mockReporter{terminal: true}
		getVersion := func(context.Context) ([]byte, error) {
			return []byte("Client Version: 3.11.0\n"), nil
		}

		runVerifyOC(context.Background(), reporter, getVersion)

		Expect(reporter.warns).To(HaveLen(2), "expected version and unsupported warnings for non-4.x oc")
		Expect(reporter.warns[0]).To(ContainSubstring("3.11.0"), "expected first warning to include the detected version")
		Expect(reporter.warns[1]).To(ContainSubstring("not supported"), "expected unsupported-version warning text")
	})

	It("Reports an error when oc returns output and an execution error", func() {
		reporter := &mockReporter{terminal: true}
		getVersion := func(context.Context) ([]byte, error) {
			return []byte("Client Version: 4.14.0\n"), fmt.Errorf("permission denied")
		}

		runVerifyOC(context.Background(), reporter, getVersion)

		Expect(reporter.errs).To(HaveLen(1), "expected one error when oc returns a non-not-found execution failure")
		Expect(reporter.errs[0]).To(ContainSubstring("permission denied"), "expected the execution error to be surfaced")
	})

	It("Does not print verifying message when not in terminal mode", func() {
		reporter := &mockReporter{terminal: false}
		getVersion := func(context.Context) ([]byte, error) {
			return []byte("Client Version: 4.14.0\n"), nil
		}

		runVerifyOC(context.Background(), reporter, getVersion)

		Expect(reporter.infos).To(BeEmpty(), "expected no informational output in non-terminal mode")
	})

	It("Rejects extra arguments via cobra.NoArgs", func() {
		Expect(Cmd.Args).NotTo(BeNil(), "expected cobra.NoArgs validation to be configured")
		err := Cmd.Args(Cmd, []string{"unexpected"})
		Expect(err).To(HaveOccurred(), "expected extra arguments to be rejected")
	})
})
