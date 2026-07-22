package input

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

func TestInput(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Input Suite")
}

var _ = Describe("Input", func() {
	Describe("UnmarshalInputFile", func() {
		It("loads a YAML file into a map", func() {
			tempDir, err := os.MkdirTemp("", "input-yaml-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			path := filepath.Join(tempDir, "spec.yaml")
			Expect(os.WriteFile(path, []byte("name: demo\ncount: 3\nnested:\n  enabled: true\n"), 0o600)).To(Succeed())

			result, err := UnmarshalInputFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKeyWithValue("name", "demo"))
			Expect(result).To(HaveKey("nested"))
		})

		It("loads a JSON file because YAML unmarshalling accepts JSON", func() {
			tempDir, err := os.MkdirTemp("", "input-json-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			path := filepath.Join(tempDir, "spec.json")
			Expect(os.WriteFile(path, []byte(`{"name":"demo","enabled":true}`), 0o600)).To(Succeed())

			result, err := UnmarshalInputFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveKeyWithValue("name", "demo"))
			Expect(result).To(HaveKeyWithValue("enabled", BeTrue()))
		})

		It("returns an error for a missing file", func() {
			_, err := UnmarshalInputFile(filepath.Join(os.TempDir(), "does-not-exist.yaml"))
			Expect(err).To(HaveOccurred())
		})

		It("returns an error for invalid YAML", func() {
			tempDir, err := os.MkdirTemp("", "input-invalid-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			path := filepath.Join(tempDir, "bad.yaml")
			Expect(os.WriteFile(path, []byte("name: [unterminated"), 0o600)).To(Succeed())

			_, err = UnmarshalInputFile(path)
			Expect(err).To(HaveOccurred())
		})

		It("returns no error for an empty file", func() {
			tempDir, err := os.MkdirTemp("", "input-empty-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			path := filepath.Join(tempDir, "empty.yaml")
			Expect(os.WriteFile(path, []byte(""), 0o600)).To(Succeed())

			result, err := UnmarshalInputFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("returns an error when the file can't be read due to permissions", func() {
			if os.Geteuid() == 0 {
				Skip("permission checks are not reliable when running as root")
			}

			tempDir, err := os.MkdirTemp("", "input-perms-*")
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				Expect(os.RemoveAll(tempDir)).To(Succeed())
			}()

			path := filepath.Join(tempDir, "unreadable.yaml")
			Expect(os.WriteFile(path, []byte("name: demo\n"), 0o600)).To(Succeed())
			Expect(os.Chmod(path, 0o000)).To(Succeed())

			_, err = UnmarshalInputFile(path)
			Expect(err).To(HaveOccurred())
			Expect(os.IsPermission(err)).To(BeTrue())
		})

		It("returns an error when the path is a directory", func() {
			tempDir, err := os.MkdirTemp("", "input-directory-*")
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				Expect(os.RemoveAll(tempDir)).To(Succeed())
			}()

			_, err = UnmarshalInputFile(tempDir)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("CheckIfHypershiftClusterOrExit", func() {
		var previousExitFunc func(int)

		BeforeEach(func() {
			previousExitFunc = exitFunc
		})

		AfterEach(func() {
			exitFunc = previousExitFunc
		})

		It("returns without exiting for a hypershift cluster", func() {
			cluster, err := cmv1.NewCluster().
				Hypershift(cmv1.NewHypershift().Enabled(true)).
				Build()
			Expect(err).NotTo(HaveOccurred())

			Expect(func() {
				CheckIfHypershiftClusterOrExit(&rosa.Runtime{}, cluster)
			}).NotTo(Panic())
		})

		It("exits when the cluster isn't a hypershift cluster", func() {
			t := test.NewTestRuntime()
			cluster, err := cmv1.NewCluster().
				Hypershift(cmv1.NewHypershift().Enabled(false)).
				Build()
			Expect(err).NotTo(HaveOccurred())

			exitFunc = func(code int) {
				panic(code)
			}

			Expect(func() {
				CheckIfHypershiftClusterOrExit(t.RosaRuntime, cluster)
			}).To(PanicWith(1))
		})
	})
})
