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

package main

import (
	"fmt"
	"os"

	semver "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/completion"
	"github.com/openshift/rosa/cmd/create"
	"github.com/openshift/rosa/cmd/describe"
	"github.com/openshift/rosa/cmd/dlt"
	"github.com/openshift/rosa/cmd/docs"
	"github.com/openshift/rosa/cmd/download"
	"github.com/openshift/rosa/cmd/edit"
	"github.com/openshift/rosa/cmd/grant"
	"github.com/openshift/rosa/cmd/hibernate"
	"github.com/openshift/rosa/cmd/initialize"
	"github.com/openshift/rosa/cmd/install"
	"github.com/openshift/rosa/cmd/link"
	"github.com/openshift/rosa/cmd/list"
	"github.com/openshift/rosa/cmd/login"
	"github.com/openshift/rosa/cmd/logout"
	"github.com/openshift/rosa/cmd/logs"
	"github.com/openshift/rosa/cmd/resume"
	"github.com/openshift/rosa/cmd/revoke"
	"github.com/openshift/rosa/cmd/uninstall"
	"github.com/openshift/rosa/cmd/upgrade"
	"github.com/openshift/rosa/cmd/verify"
	"github.com/openshift/rosa/cmd/version"
	"github.com/openshift/rosa/cmd/whoami"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/reporter"
)

var root = &cobra.Command{
	Use:   "rosa",
	Short: "Command line tool for ROSA.",
	Long:  "Command line tool for Red Hat OpenShift Service on AWS.",
}

func init() {
	// Add the command line flags:
	fs := root.PersistentFlags()
	arguments.AddDebugFlag(fs)

	CheckROSACliVersion()

	// Register the subcommands:
	root.AddCommand(completion.Cmd)
	root.AddCommand(create.Cmd)
	root.AddCommand(describe.Cmd)
	root.AddCommand(dlt.Cmd)
	root.AddCommand(docs.Cmd)
	root.AddCommand(download.Cmd)
	root.AddCommand(edit.Cmd)
	root.AddCommand(grant.Cmd)
	root.AddCommand(list.Cmd)
	root.AddCommand(initialize.Cmd)
	root.AddCommand(install.Cmd)
	root.AddCommand(login.Cmd)
	root.AddCommand(logout.Cmd)
	root.AddCommand(logs.Cmd)
	root.AddCommand(revoke.Cmd)
	root.AddCommand(uninstall.Cmd)
	root.AddCommand(upgrade.Cmd)
	root.AddCommand(verify.Cmd)
	root.AddCommand(version.Cmd)
	root.AddCommand(whoami.Cmd)
	root.AddCommand(hibernate.Cmd)
	root.AddCommand(resume.Cmd)
	root.AddCommand(link.Cmd)
}

func main() {
	// Execute the root command:
	root.SetArgs(os.Args[1:])
	err := root.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute root command: %s\n", err)
		os.Exit(1)
	}
}

func CheckROSACliVersion() {
	reporter := reporter.CreateReporterOrExit()
	genericErrorMessage := "We could not determine the latest ROSA cli version. Goto " +
		"'https://console.redhat.com/openshift/create/rosa/welcome' if you wish to check and update your client"
	latestTag, err := ocm.GetLatestROSACliVersion()
	if err != nil {
		reporter.Debugf("%s", err)
		reporter.Infof("%s", genericErrorMessage)
		return
	}
	if latestTag != "" && info.Version != latestTag {
		latestVersion, err := semver.NewVersion(latestTag)
		if err != nil {
			reporter.Debugf("%s", err)
			reporter.Infof("%s", genericErrorMessage)
			return
		}
		currentVersion, err := semver.NewVersion(info.Version)
		if err != nil {
			reporter.Debugf("%s", err)
			reporter.Infof("%s", genericErrorMessage)
			return
		}
		if currentVersion.LessThanOrEqual(latestVersion) {
			reporter.Infof("A newer version of rosa cli '%s' is available for download. Goto "+
				"'https://console.redhat.com/openshift/create/rosa/welcome' if you wish to update", latestTag)
		}
	}
}
