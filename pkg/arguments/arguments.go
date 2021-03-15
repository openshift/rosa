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

// This file contains functions that add common arguments to the command line.

package arguments

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openshift/rosa/pkg/aws/profile"
	"github.com/openshift/rosa/pkg/aws/region"
	"github.com/openshift/rosa/pkg/debug"
)

var hasUnknownFlags bool

// ParseUnknownFlags parses all flags from the CLI, including
// unknown ones, and adds them to the current command tree
func ParseUnknownFlags(cmd *cobra.Command, argv []string) error {
	flags := cmd.Flags()

	prevArg := ""
	for _, arg := range argv {
		// If there are two consecutive flags, assume we've already
		// dealt with the previous one by setting it as 'true'.
		if strings.HasPrefix(arg, "-") && prevArg != "" {
			var boolVal bool
			flags.BoolVar(&boolVal, prevArg, false, "")
			flags.Set(prevArg, "true")
			prevArg = ""
			hasUnknownFlags = true
		}

		switch {
		// A long flag with a space separated value
		case strings.HasPrefix(arg, "--") && !strings.Contains(arg, "="):
			arg = arg[2:]
			// Skip EOF and known flags
			if len(arg) == 0 || flags.Lookup(arg) != nil {
				continue
			}
			prevArg = arg
			continue
		// The value for the previous flag
		case prevArg != "":
			var strVal string
			flags.StringVar(&strVal, prevArg, "", "")
			flags.Set(prevArg, arg)
			prevArg = ""
			hasUnknownFlags = true
			continue
		// A long flag with an '=' separated value
		case strings.HasPrefix(arg, "--") && strings.Contains(arg, "="):
			val := strings.Split(arg[2:], "=")
			// Only consider unknown flags with values
			if len(val) == 2 && flags.Lookup(val[0]) == nil {
				var strVal string
				flags.StringVar(&strVal, val[0], "", "")
				flags.Set(val[0], val[1])
				hasUnknownFlags = true
			}
			continue
		}
	}

	return flags.Parse(argv)
}

// HasUnknownFlags returns whether the flag parser detected any unknown flags
func HasUnknownFlags() bool {
	return hasUnknownFlags
}

// AddDebugFlag adds the '--debug' flag to the given set of command line flags.
func AddDebugFlag(fs *pflag.FlagSet) {
	debug.AddFlag(fs)
}

// AddProfileFlag adds the '--profile' flag to the given set of command line flags.
func AddProfileFlag(fs *pflag.FlagSet) {
	profile.AddFlag(fs)
}

func GetProfile() string {
	return profile.Profile()
}

// AddRegionFlag adds the '--region' flag to the given set of command line flags.
func AddRegionFlag(fs *pflag.FlagSet) {
	region.AddFlag(fs)
}

func GetRegion() string {
	return region.Region()
}
