package hyperfleet

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	platformAPIAnnotation = "platform-api-group"
	sharedFlagAnnotation  = "platform-api-shared"
)

// RegisterAndMarkPlatformAPIFlags snapshots pre-existing flags, calls register
// (which should invoke Register*Flags), then classifies newly registered flags:
//   - Flags in platformAPIFlags that were NEW → tagged as Platform API-only
//   - Flags in platformAPIFlags that already EXISTED → tagged as Shared (used by both paths)
//
// This is the single entry point; call AddPlatformAPIFlagSection once after all
// Register*Flags calls to install the custom help function.
func RegisterAndMarkPlatformAPIFlags(cmd *cobra.Command, register func(), platformAPIFlags []string) {
	preExisting := map[string]bool{}
	cmd.Flags().VisitAll(func(f *pflag.Flag) { preExisting[f.Name] = true })

	register()

	for _, name := range platformAPIFlags {
		if !preExisting[name] {
			// Newly registered by HF — Platform API only.
			if cmd.Flags().Lookup(name) != nil {
				_ = cmd.Flags().SetAnnotation(name, platformAPIAnnotation, []string{"true"})
			}
		} else {
			// Pre-existing (also registered by OCM) — Shared.
			if cmd.Flags().Lookup(name) != nil {
				_ = cmd.Flags().SetAnnotation(name, sharedFlagAnnotation, []string{"true"})
			}
		}
	}
}

// AddPlatformAPIFlagSection installs a custom help function on cmd that renders
// shared and Platform API-only flags in dedicated sections before "Global Flags:".
func AddPlatformAPIFlagSection(cmd *cobra.Command) {
	defaultHelp := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		originalOut := c.OutOrStdout()

		// Temporarily hide annotated flags from the standard section.
		c.Flags().VisitAll(func(f *pflag.Flag) {
			_, isPlatformAPI := f.Annotations[platformAPIAnnotation]
			_, isShared := f.Annotations[sharedFlagAnnotation]
			if isPlatformAPI || isShared {
				f.Hidden = true
			}
		})

		var buf bytes.Buffer
		c.SetOut(&buf)
		defaultHelp(c, args)
		c.SetOut(originalOut)

		// Restore and collect flags by group.
		var platformFlags, sharedFlags []*pflag.Flag
		c.Flags().VisitAll(func(f *pflag.Flag) {
			if _, ok := f.Annotations[platformAPIAnnotation]; ok {
				f.Hidden = false
				platformFlags = append(platformFlags, f)
			} else if _, ok := f.Annotations[sharedFlagAnnotation]; ok {
				f.Hidden = false
				sharedFlags = append(sharedFlags, f)
			}
		})

		output := buf.String()

		buildSection := func(title string, flags []*pflag.Flag) string {
			if len(flags) == 0 {
				return ""
			}
			fs := pflag.NewFlagSet(title, pflag.ContinueOnError)
			for _, f := range flags {
				fs.AddFlag(f)
			}
			return "\n" + title + ":\n" + fs.FlagUsages()
		}

		// Insert both sections just before "Global Flags:".
		extra := buildSection("Shared flags (OCM v1 and Platform API)", sharedFlags) +
			buildSection("Platform API flags", platformFlags)

		if extra != "" {
			if idx := strings.Index(output, "\nGlobal Flags:"); idx >= 0 {
				output = output[:idx] + extra + output[idx:]
			} else {
				output += extra
			}
		}

		_, _ = fmt.Fprint(originalOut, output)
	})
}
