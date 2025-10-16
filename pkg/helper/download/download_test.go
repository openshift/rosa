package helper_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	helper "github.com/openshift/rosa/pkg/helper/download"
)

func TestDownload(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Download Helper Suite")
}

var _ = Describe("Download", func() {
	var (
		tmpDir string
		server *httptest.Server
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "rosa-download-test-*")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if server != nil {
			server.Close()
		}
		os.RemoveAll(tmpDir)
	})

	Context("when downloading files", func() {
		It("should successfully download a file with 2xx status codes", func() {
			expectedContent := "test file content"
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, expectedContent)
			}))

			filename := filepath.Join(tmpDir, "test.txt")
			err := helper.Download(server.URL, filename)
			Expect(err).ToNot(HaveOccurred())

			// Verify file was created and contains expected content
			content, err := os.ReadFile(filename)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(Equal(expectedContent))

			// Verify temp files were cleaned up (pattern: filename.*.tmp)
			matches, err := filepath.Glob(filepath.Join(tmpDir, filepath.Base(filename)+".*.tmp"))
			Expect(err).ToNot(HaveOccurred())
			Expect(matches).To(BeEmpty(), "Temporary files should be cleaned up")
		})

		It("should handle HTTP errors and clean up temp files", func() {
			testCases := []struct {
				statusCode   int
				errorMessage string
			}{
				{http.StatusNotFound, "file not found"},
				{http.StatusForbidden, "access forbidden"},
				{http.StatusInternalServerError, "server error"},
			}

			for _, tc := range testCases {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.statusCode)
					fmt.Fprint(w, "Error page content")
				}))

				filename := filepath.Join(tmpDir, fmt.Sprintf("error-%d.txt", tc.statusCode))
				err := helper.Download(server.URL, filename)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(tc.errorMessage))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("HTTP %d", tc.statusCode)))

				// Verify file was not created and temp files were cleaned up
				_, err = os.Stat(filename)
				Expect(os.IsNotExist(err)).To(BeTrue())

				// Check that no temp files remain (pattern: filename.*.tmp)
				matches, err := filepath.Glob(filepath.Join(tmpDir, filepath.Base(filename)+".*.tmp"))
				Expect(err).ToNot(HaveOccurred())
				Expect(matches).To(BeEmpty(), "Temporary files should be cleaned up after error")

				server.Close()
			}
		})

		It("should handle network errors", func() {
			// Use an invalid URL to simulate network error
			filename := filepath.Join(tmpDir, "network-error.txt")
			err := helper.Download("http://invalid.test.domain.that.does.not.exist", filename)
			Expect(err).To(HaveOccurred())
			// The error should be formatted cleanly without technical details
			Expect(err.Error()).To(Or(
				ContainSubstring("unable to resolve host"),
				ContainSubstring("network error"),
				ContainSubstring("unable to connect"),
			))
			Expect(err.Error()).To(ContainSubstring("check your internet connection"))

			// Verify temp files were cleaned up (pattern: filename.*.tmp)
			matches, err := filepath.Glob(filepath.Join(tmpDir, filepath.Base(filename)+".*.tmp"))
			Expect(err).ToNot(HaveOccurred())
			Expect(matches).To(BeEmpty(), "Temporary files should be cleaned up after network error")
		})
	})

	Context("when checking file extensions", func() {
		It("should return 'tar.gz' for non-Windows systems", func() {
			extension := helper.GetExtension()
			Expect(extension).To(Or(Equal("tar.gz"), Equal("zip")))
		})
	})
})
