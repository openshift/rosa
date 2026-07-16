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

	verifyOC "github.com/openshift/rosa/cmd/verify/oc"
	helper "github.com/openshift/rosa/pkg/helper/download"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var downloadFn = helper.Download
var verifyOCRun = verifyOC.Cmd.Run

var Cmd = &cobra.Command{
	Use:     "openshift-client",
	Aliases: []string{"oc", "openshift"},
	Short:   "Download OpenShift client tools",
	Long:    "Downloads to latest compatible version of the OpenShift client tools.",
	Example: `  # Download oc client tools
  rosa download oc`,
	Run:  run,
	Args: cobra.NoArgs,
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporter()
	err := runDownloadOC(reporter, verifyOCRun, downloadFn, runtime.GOOS, cmd, argv)
	if err != nil {
		os.Exit(1)
	}
}

func runDownloadOC(
	reporter rprtr.Logger,
	verify func(*cobra.Command, []string),
	download func(string, string) error,
	goos string,
	cmd *cobra.Command,
	argv []string,
) error {
	verify(cmd, argv)

	platform := platformForGOOS(goos)
	extension := extensionForGOOS(goos)

	filename := fmt.Sprintf("openshift-client-%s.%s", platform, extension)
	downloadURL := fmt.Sprintf("https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest/%s", filename)

	reporter.Infof("Downloading %s", downloadURL)

	err := download(downloadURL, filename)
	if err != nil {
		reporter.Errorf("%s", err)
		return err
	}

	reporter.Infof("Successfully downloaded %s", filename)
	return nil
}

func platformForGOOS(goos string) string {
	if goos == "darwin" {
		return "mac"
	}
	return goos
}

func extensionForGOOS(goos string) string {
	if goos == "windows" {
		return "zip"
	}
	return "tar.gz"
}
