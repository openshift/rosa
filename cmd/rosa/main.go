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
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/attach"
	"github.com/openshift/rosa/cmd/completion"
	"github.com/openshift/rosa/cmd/config"
	"github.com/openshift/rosa/cmd/create"
	"github.com/openshift/rosa/cmd/describe"
	"github.com/openshift/rosa/cmd/detach"
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
	"github.com/openshift/rosa/cmd/register"
	"github.com/openshift/rosa/cmd/resume"
	"github.com/openshift/rosa/cmd/revoke"
	"github.com/openshift/rosa/cmd/token"
	"github.com/openshift/rosa/cmd/uninstall"
	"github.com/openshift/rosa/cmd/unlink"
	"github.com/openshift/rosa/cmd/upgrade"
	"github.com/openshift/rosa/cmd/verify"
	"github.com/openshift/rosa/cmd/version"
	"github.com/openshift/rosa/cmd/whoami"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/color"
	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/reporter"
	versionUtils "github.com/openshift/rosa/pkg/version"
)

var root = &cobra.Command{
	Use:   "rosa",
	Short: "Command line tool for ROSA.",
	Long: "Command line tool for Red Hat OpenShift Service on AWS.\n" +
		"For further documentation visit " +
		"https://access.redhat.com/documentation/en-us/red_hat_openshift_service_on_aws\n",
	PersistentPreRun: versionCheck,
	Args:             cobra.NoArgs,
}

func init() {
	// Add the command line flags:
	fs := root.PersistentFlags()
	color.AddFlag(root)
	arguments.AddDebugFlag(fs)

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
	root.AddCommand(register.Cmd)
	root.AddCommand(revoke.Cmd)
	root.AddCommand(uninstall.Cmd)
	root.AddCommand(upgrade.Cmd)
	root.AddCommand(verify.Cmd)
	root.AddCommand(version.NewRosaVersionCommand())
	root.AddCommand(whoami.Cmd)
	root.AddCommand(hibernate.GenerateCommand())
	root.AddCommand(resume.GenerateCommand())
	root.AddCommand(link.Cmd)
	root.AddCommand(unlink.Cmd)
	root.AddCommand(token.Cmd)
	root.AddCommand(config.Cmd)
	root.AddCommand(attach.NewRosaAttachCommand())
	root.AddCommand(detach.NewRosaDetachCommand())
}

func main() {
	// Execute the root command:
	root.SetArgs(os.Args[1:])
	err := root.Execute()
	if err != nil {
		if !strings.Contains(err.Error(), "Did you mean this?") {
			fmt.Fprintf(os.Stderr, "Failed to execute root command: %s\n", err)
		}
		os.Exit(1)
	}
}

func versionCheck(cmd *cobra.Command, _ []string) {
	if !versionUtils.ShouldRunCheck(cmd) {
		return
	}

	rprtr := reporter.CreateReporter()
	rosaVersion, err := versionUtils.NewRosaVersion()
	if err != nil {
		rprtr.Debugf("Could not verify the current version of ROSA: %v", err)
		rprtr.Debugf("You might be running on an outdated version. Make sure you are using the current version of ROSA.")
		return
	}
	latestVersionFromMirror, isLatest, err := rosaVersion.IsLatest(info.DefaultVersion)
	if err != nil {
		rprtr.Debugf("There was a problem retrieving the latest version of ROSA: %v", err)
		rprtr.Debugf("You might be running on an outdated version. Make sure you are using the current version of ROSA.")
		return
	}
	if !isLatest {
		rprtr.Warnf("The current version (%s) is not up to date with latest rosa cli released version (%s).",
			info.DefaultVersion,
			latestVersionFromMirror.Original(),
		)
		rprtr.Warnf("It is recommended that you update to the latest version.")
	}
}
