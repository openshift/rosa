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

package docs

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	dir    string
	format string
}

var Cmd = &cobra.Command{
	Use:    "docs",
	Short:  "Generates documentation files",
	Hidden: true,
	Run:    run,
	Args:   cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.dir,
		"dir",
		"d",
		"./docs",
		"The directory where to save the documentation to",
	)

	flags.StringVarP(
		&args.format,
		"format",
		"f",
		"markdown",
		"The output format of the documentation. Valid options are 'markdown', 'man', 'restructured'",
	)
}

func run(cmd *cobra.Command, _ []string) {
	cmd.Root().DisableAutoGenTag = true
	var err error

	switch args.format {
	case "markdown":
		err = doc.GenMarkdownTree(cmd.Root(), args.dir)
	case "man":
		year := time.Now().Year()
		header := &doc.GenManHeader{
			Title:   "ROSA",
			Section: "1",
			Source:  fmt.Sprintf("Copyright (c) %d Red Hat, Inc.", year),
		}
		err = doc.GenManTree(cmd.Root(), header, args.dir)
	case "restructured":
		err = doc.GenReSTTree(cmd.Root(), args.dir)
	}

	r := rosa.NewRuntime()
	if err != nil {
		r.Reporter.Errorf("Failed to generate documents: %v", err)
		os.Exit(1)
	}

	r.Reporter.Infof("Documents generated successfully on '%s'", args.dir)
}
