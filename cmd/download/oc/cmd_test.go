package oc

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

type mockReporter struct {
	infos []string
	errs  []string
}

func (m *mockReporter) Debugf(_ string, _ ...any) {}
func (m *mockReporter) Infof(f string, a ...any)  { m.infos = append(m.infos, fmt.Sprintf(f, a...)) }
func (m *mockReporter) Warnf(_ string, _ ...any)  {}
func (m *mockReporter) Errorf(f string, a ...any) error {
	m.errs = append(m.errs, fmt.Sprintf(f, a...))
	return nil
}
func (m *mockReporter) IsTerminal() bool { return true }

var _ = Describe("download oc", func() {
	It("Calls verify preflight before downloading", func() {
		verifyCalled := false
		verify := func(_ *cobra.Command, _ []string) {
			verifyCalled = true
		}
		download := func(_, _ string) error { return nil }
		reporter := &mockReporter{}
		cmd := &cobra.Command{}

		err := runDownloadOC(reporter, verify, download, "linux", cmd, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(verifyCalled).To(BeTrue())
	})

	It("Downloads successfully with correct URL for linux", func() {
		var capturedURL, capturedFile string
		verify := func(_ *cobra.Command, _ []string) {}
		download := func(url, file string) error {
			capturedURL = url
			capturedFile = file
			return nil
		}
		reporter := &mockReporter{}
		cmd := &cobra.Command{}

		err := runDownloadOC(reporter, verify, download, "linux", cmd, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(capturedURL).To(ContainSubstring("openshift-client-linux.tar.gz"))
		Expect(capturedFile).To(Equal("openshift-client-linux.tar.gz"))
		Expect(reporter.infos).To(HaveLen(2))
		Expect(reporter.infos[1]).To(ContainSubstring("Successfully downloaded"))
	})

	It("Builds a windows zip target independent of host OS", func() {
		var capturedURL, capturedFile string
		verify := func(_ *cobra.Command, _ []string) {}
		download := func(url, file string) error {
			capturedURL = url
			capturedFile = file
			return nil
		}
		reporter := &mockReporter{}
		cmd := &cobra.Command{}

		err := runDownloadOC(reporter, verify, download, "windows", cmd, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(capturedURL).To(ContainSubstring("openshift-client-windows.zip"))
		Expect(capturedFile).To(Equal("openshift-client-windows.zip"))
	})

	It("Returns error when download fails", func() {
		verify := func(_ *cobra.Command, _ []string) {}
		download := func(_, _ string) error { return fmt.Errorf("network timeout") }
		reporter := &mockReporter{}
		cmd := &cobra.Command{}

		err := runDownloadOC(reporter, verify, download, "linux", cmd, nil)
		Expect(err).To(HaveOccurred())
		Expect(reporter.errs).To(HaveLen(1))
		Expect(reporter.errs[0]).To(ContainSubstring("network timeout"))
	})

	It("Maps darwin to mac for platform detection", func() {
		Expect(platformForGOOS("darwin")).To(Equal("mac"))
		Expect(platformForGOOS("linux")).To(Equal("linux"))
		Expect(platformForGOOS("windows")).To(Equal("windows"))
	})

	It("Maps windows to zip extension", func() {
		Expect(extensionForGOOS("windows")).To(Equal("zip"))
		Expect(extensionForGOOS("linux")).To(Equal("tar.gz"))
	})

	It("Rejects extra arguments via cobra.NoArgs", func() {
		Expect(Cmd.Args).NotTo(BeNil())
		err := Cmd.Args(Cmd, []string{"unexpected"})
		Expect(err).To(HaveOccurred())
	})

	It("Has expected command aliases", func() {
		Expect(Cmd.Aliases).To(ContainElement("oc"))
		Expect(Cmd.Aliases).To(ContainElement("openshift"))
	})
})
