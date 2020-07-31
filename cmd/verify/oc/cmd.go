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

package oc

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	rprtr "github.com/openshift/moactl/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:     "openshift-client",
	Aliases: []string{"oc", "openshift"},
	Short:   "Verify OpenShift client tools",
	Long:    "Verify that the OpenShift client tools is installed and compatible.",
	Example: `  # Verify oc client tools
  moactl verify oc`,
	Run: run,
}

func run(_ *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()

	// Verify whether `oc` is installed
	reporter.Infof("Verifying whether OpenShift command-line tool is available...")
	ocDownloadURL := "https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest"

	output, err := exec.Command("oc", "version").Output()
	if err != nil {
		reporter.Errorf("OpenShift command-line tool is not installed.\n"+
			"Go to %s to download the OpenShift client and add it to your PATH.", ocDownloadURL)
		os.Exit(1)
	}

	// Parse the version for the OpenShift Client
	version := strings.Replace(strings.Split(string(output), "\n")[0], "\n", "", 1)
	isCorrectVersion, err := regexp.Match(`\W4.\d*`, output)
	if err != nil {
		reporter.Errorf("Failed to parse OpenShift Client version: %v", err)
		os.Exit(1)
	}

	if !isCorrectVersion {
		reporter.Warnf("Current OpenShift %s", version)
		reporter.Warnf("Your version of the OpenShift command-line tool is not supported.")
		fmt.Printf("Go to %s to download the latest version.\n", ocDownloadURL)
		os.Exit(1)
	}

	reporter.Infof("Current OpenShift %s", version)
}
