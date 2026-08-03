package hyperfleet

import "github.com/spf13/cobra"

var hyperfleetURL string

func AddFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(
		&hyperfleetURL,
		"hyperfleet-url",
		"",
		"Platform API v2 endpoint URL. When set, commands route to the Platform API instead of OCM.",
	)
}

func Enabled() bool       { return hyperfleetURL != "" }
func ExplicitURL() string { return hyperfleetURL }
