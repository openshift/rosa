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

// SetURL seeds the hyperfleet URL from an external source (e.g. stored config)
// when the --hyperfleet-url flag was not passed explicitly. A flag value always
// takes precedence: if Enabled() is already true, this is a no-op.
func SetURL(url string) {
	if hyperfleetURL == "" {
		hyperfleetURL = url
	}
}

// Reset clears the hyperfleet URL back to the zero value.
func Reset() { hyperfleetURL = "" }
