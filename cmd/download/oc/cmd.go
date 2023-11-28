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
	"runtime"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/verify/oc"
	helper "github.com/openshift/rosa/pkg/helper/download"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:     "openshift-client",
	Aliases: []string{"oc", "openshift"},
	Short:   "Download OpenShift client tools",
	Long:    "Downloads to latest compatible version of the OpenShift client tools.",
	Example: `  # Download oc client tools
  rosa download oc`,
	Run: run,
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()

	// Verify whether `oc` is installed
	oc.Cmd.Run(cmd, argv)

	platform := getPlatform()
	extension := helper.GetExtension()

	filename := fmt.Sprintf("openshift-client-%s.%s", platform, extension)
	downloadURL := fmt.Sprintf("https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest/%s", filename)

	reporter.Infof("Downloading %s", downloadURL)

	err := helper.Download(downloadURL, filename)
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	reporter.Infof("Successfully downloaded %s", filename)
}

// Get the platform name used on the oc tarball filename
func getPlatform() string {
	if runtime.GOOS == "darwin" {
		return "mac"
	}
	return runtime.GOOS
}
