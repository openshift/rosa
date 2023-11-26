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
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:     "rosa-client",
	Aliases: []string{"rosa"},
	Short:   "Verify ROSA client tools",
	Long:    "Verify that the ROSA client tools is installed and compatible.",
	Example: `  # Verify rosa client tools
  rosa verify rosa`,
	Run: run,
}

const (
	DownloadLatestMirrorFolder = "https://mirror.openshift.com/pub/openshift-v4/clients/rosa/latest/"
	baseReleasesFolder         = "https://mirror.openshift.com/pub/openshift-v4/clients/rosa/"
	consoleLatestFolder        = "https://console.redhat.com/openshift/downloads#tool-rosa"
)

func run(_ *cobra.Command, _ []string) {
	rprtr := reporter.CreateReporterOrExit()

	currVersion, err := version.NewVersion(info.Version)
	if err != nil {
		rprtr.Errorf("There was a problem retrieving current version: %s", err)
		os.Exit(1)
	}
	latestVersionFromMirror, err := retrieveLatestVersionFromMirror()
	if err != nil {
		rprtr.Errorf("There was a problem retrieving latest version from mirror: %s", err)
		os.Exit(1)
	}
	if currVersion.LessThan(latestVersionFromMirror) {
		rprtr.Infof(
			"There is a newer release version '%s', please consider updating: %s",
			latestVersionFromMirror, consoleLatestFolder,
		)
	} else if rprtr.IsTerminal() {
		rprtr.Infof("Your ROSA CLI is up to date.")
	}
	os.Exit(0)
}

func retrievePossibleVersionsFromMirror() ([]string, error) {
	resp, err := http.Get(baseReleasesFolder)
	if err != nil {
		return []string{}, weberr.Wrapf(err, "Error setting up request for latest released rosa cli")
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		return []string{},
			weberr.Errorf("Error while requesting latest released rosa cli: %d %s", resp.StatusCode, resp.Status)
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return []string{}, weberr.Wrapf(err, "Error parsing response body")
	}
	possibleVersions := []string{}
	doc.Find(".file").Each(func(i int, s *goquery.Selection) {
		s.Find("a").Each(func(j int, ss *goquery.Selection) {
			if version, ok := ss.Attr("href"); ok {
				version = strings.TrimSpace(version)
				version = strings.TrimRight(version, "/")
				if version != "latest" {
					possibleVersions = append(possibleVersions, version)
				}
			}
		})
	})
	return possibleVersions, nil
}

func retrieveLatestVersionFromMirror() (*version.Version, error) {
	possibleVersions, err := retrievePossibleVersionsFromMirror()
	if err != nil {
		return nil, weberr.Wrapf(err, "There was a problem retrieving possible versions from mirror.")
	}
	if len(possibleVersions) == 0 {
		return nil, weberr.Errorf("No versions available in mirror %s", baseReleasesFolder)
	}
	latestVersion, err := version.NewVersion(possibleVersions[0])
	if err != nil {
		return nil, weberr.Wrapf(err, "There was a problem retrieving latest version.")
	}
	for _, ver := range possibleVersions[1:] {
		curVersion, err := version.NewVersion(ver)
		if err != nil {
			continue
		}
		if curVersion.GreaterThan(latestVersion) {
			latestVersion = curVersion
		}
	}
	return latestVersion, nil
}
