/*
Copyright (c) 2023 Red Hat, Inc.

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

package rosa

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	helper "github.com/openshift/rosa/pkg/helper/download"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/version"
)

var Cmd = &cobra.Command{
	Use:     "rosa-client",
	Aliases: []string{"rosa"},
	Short:   "Download ROSA client tools",
	Long:    "Downloads to latest compatible version of the ROSA client tools.",
	Example: `  # Download rosa client tools
  rosa download rosa`,
	Run:  run,
	Args: cobra.NoArgs,
}

func run(_ *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporter()

	platform := getPlatform()
	extension := helper.GetExtension()

	filename := fmt.Sprintf("rosa-%s.%s", platform, extension)

	downloadURL := fmt.Sprintf("%s%s", version.DownloadLatestMirrorFolder, filename)

	reporter.Infof("Downloading %s to your current directory", downloadURL)

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
		return "macosx"
	}
	return runtime.GOOS
}
