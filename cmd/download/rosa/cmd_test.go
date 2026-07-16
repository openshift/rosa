package rosa

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/version"
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

var _ = Describe("download rosa", func() {
	It("Downloads successfully with correct URL using DownloadLatestMirrorFolder", func() {
		var capturedURL, capturedFile string
		download := func(url, file string) error {
			capturedURL = url
			capturedFile = file
			return nil
		}
		reporter := &mockReporter{}

		err := runDownloadRosa(reporter, download, "linux")
		Expect(err).NotTo(HaveOccurred())
		Expect(capturedURL).To(HavePrefix(version.DownloadLatestMirrorFolder))
		Expect(capturedURL).To(ContainSubstring("rosa-linux.tar.gz"))
		Expect(capturedFile).To(Equal("rosa-linux.tar.gz"))
		Expect(reporter.infos).To(HaveLen(2))
		Expect(reporter.infos[1]).To(ContainSubstring("Successfully downloaded"))
	})

	It("Builds a windows zip target independent of host OS", func() {
		var capturedURL, capturedFile string
		download := func(url, file string) error {
			capturedURL = url
			capturedFile = file
			return nil
		}
		reporter := &mockReporter{}

		err := runDownloadRosa(reporter, download, "windows")
		Expect(err).NotTo(HaveOccurred())
		Expect(capturedURL).To(HavePrefix(version.DownloadLatestMirrorFolder))
		Expect(capturedURL).To(ContainSubstring("rosa-windows.zip"))
		Expect(capturedFile).To(Equal("rosa-windows.zip"))
	})

	It("Returns error when download fails", func() {
		download := func(_, _ string) error { return fmt.Errorf("connection refused") }
		reporter := &mockReporter{}

		err := runDownloadRosa(reporter, download, "linux")
		Expect(err).To(HaveOccurred())
		Expect(reporter.errs).To(HaveLen(1))
		Expect(reporter.errs[0]).To(ContainSubstring("connection refused"))
	})

	It("Maps darwin to macosx for platform detection", func() {
		Expect(platformForGOOS("darwin")).To(Equal("macosx"))
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
		Expect(Cmd.Aliases).To(ContainElement("rosa"))
	})
})
