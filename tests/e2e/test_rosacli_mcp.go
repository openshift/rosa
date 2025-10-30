/*
Copyright (c) 2020 Red Hat, Inc.

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

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/log"
)

var _ = Describe("MCP Server",
	labels.Feature.Cluster,
	func() {
		defer GinkgoRecover()

		It("can list all MCP tools and resources - [id:mcp-003]",
			labels.Medium, labels.Runtime.OCMResources,
			func() {
				By("Finding the test script")
				repoRoot, err := findRepoRoot()
				Expect(err).ToNot(HaveOccurred(), "Failed to find repo root")

				testScript := filepath.Join(repoRoot, "hack", "test_mcp.py")
				Expect(testScript).To(BeAnExistingFile(), "test_mcp.py should exist")

				By("Running MCP test script")
				cmd := exec.Command("python3", testScript)
				cmd.Dir = repoRoot
				cmd.Env = os.Environ()

				output, err := cmd.CombinedOutput()
				log.Logger.Infof("MCP test script output:\n%s", string(output))

				Expect(err).ToNot(HaveOccurred(), "MCP test script should execute successfully")
			},
		)
	})

// findRepoRoot finds the repository root by looking for go.mod file
func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	repoRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
			return repoRoot, nil
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			// Reached filesystem root without finding go.mod
			return "", fmt.Errorf("could not find repo root (go.mod)")
		}
		repoRoot = parent
	}
}
