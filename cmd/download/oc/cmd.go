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
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	rprtr "github.com/openshift/moactl/pkg/reporter"

	"github.com/openshift/moactl/cmd/verify/oc"
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
	extension := getExtension()

	filename := fmt.Sprintf("openshift-client-%s.%s", platform, extension)
	downloadURL := fmt.Sprintf("https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest/%s", filename)

	reporter.Infof("Downloading %s", downloadURL)

	err := download(downloadURL, filename)
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

// Get the extension used for the compressed oc file
func getExtension() string {
	if runtime.GOOS == "windows" {
		return "zip"
	}
	return "tar.gz"
}

// download will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory. We pass an io.TeeReader
// into Copy() to report progress on the download.
func download(url string, filename string) error {
	// Create the file, but give it a tmp file extension, this means we won't overwrite a
	// file until it's downloaded, but we'll remove the tmp extension once downloaded.
	out, err := os.Create(filename + ".tmp")
	if err != nil {
		return err
	}

	// Get the data
	// nolint:gosec
	resp, err := http.Get(url)
	if err != nil {
		out.Close()
		return err
	}
	defer resp.Body.Close()

	// Create our progress reporter and pass it to be used alongside our writer
	counter := &WriteCounter{}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		out.Close()
		return err
	}

	// The progress use the same line so print a new line once it's finished downloading
	fmt.Print("\n")

	// Close the file without defer so it can happen before Rename()
	out.Close()

	if err = os.Rename(filename+".tmp", filename); err != nil {
		return err
	}
	return nil
}

// WriteCounter counts the number of bytes written to it. It implements to the io.Writer interface
// and we can pass this into io.TeeReader() which will report progress on each write cycle.
type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	fmt.Printf("\r%s", strings.Repeat(" ", 35))

	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
}
