// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package occli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/config"
)

func TestOCClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OC Client Suite")
}

var _ = Describe("IDP admin runtime state", func() {
	It("keeps credentials and kubeconfigs out of artifacts across test steps", func() {
		const credentials = "dummy-admin:dummy-password"
		switch os.Getenv("ROSA_TEST_IDP_STATE_STEP") {
		case "write":
			Expect(config.Test.ClusterIDPAdminUsernamePassword).To(Equal(
				filepath.Join(config.Test.OutputDir, ".idp-admin-username-password")))
			Expect(os.WriteFile(config.Test.ClusterIDPAdminUsernamePassword, []byte(credentials), 0600)).To(Succeed())
			Expect(os.WriteFile(config.Test.APIURLFile, []byte("https://api.example.test\n"), 0600)).To(Succeed())
			return
		case "read":
			client, err := NewOCClient()
			Expect(err).ToNot(HaveOccurred())
			Expect(filepath.Dir(client.KubePath)).To(Equal(config.Test.OutputDir))
			content, err := os.ReadFile(client.KubePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(Equal("dummy-kubeconfig"))
			info, err := os.Stat(client.KubePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Mode().Perm()).To(Equal(os.FileMode(0600)))
			return
		}

		runtimeDir := GinkgoT().TempDir()
		artifactDir := GinkgoT().TempDir()
		binDir := GinkgoT().TempDir()
		// Check the login arguments without contacting a cluster or printing credentials.
		oc := `#!/bin/sh
test "$#" -eq 8 && test "$1" = login && test "$2" = https://api.example.test &&
test "$3" = -u && test "$4" = dummy-admin &&
test "$5" = -p && test "$6" = dummy-password &&
test "$7" = --kubeconfig || exit 1
printf '%s' dummy-kubeconfig > "$8"
`
		Expect(os.WriteFile(filepath.Join(binDir, "oc"), []byte(oc), 0700)).To(Succeed())
		for _, step := range []string{"write", "read"} {
			command := exec.Command(os.Args[0], "-test.run=^TestOCClient$",
				"-ginkgo.focus=keeps credentials and kubeconfigs out of artifacts across test steps")
			command.Env = append(os.Environ(),
				"SHARED_DIR="+runtimeDir,
				"ARTIFACT_DIR="+artifactDir,
				"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
				"ROSA_TEST_IDP_STATE_STEP="+step)
			output, err := command.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "%s", output)
		}
		entries, err := os.ReadDir(artifactDir)
		Expect(err).ToNot(HaveOccurred())
		Expect(entries).To(BeEmpty())
	})

	It("uses an explicitly supplied kubeconfig without reading IDP credentials", func() {
		kubeconfigFile := filepath.Join(GinkgoT().TempDir(), "kubeconfig")
		client, err := NewOCClient(kubeconfigFile)
		Expect(err).ToNot(HaveOccurred())
		Expect(client.KubePath).To(Equal(kubeconfigFile))
	})
})
