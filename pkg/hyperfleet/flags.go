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
	_ = cmd.PersistentFlags().MarkHidden("hyperfleet-url")
}

func Enabled() bool       { return hyperfleetURL != "" }
func ExplicitURL() string { return hyperfleetURL }

// SetURL seeds the hyperfleet URL from an external source (e.g. stored config)
// when the --hyperfleet-url flag was not passed explicitly. A flag value always
// takes precedence: if Enabled() is already true, this is a no-op. Non-HTTPS
// URLs are silently rejected to prevent cleartext endpoints from being used.
func SetURL(url string) {
	if hyperfleetURL == "" && ValidateURL(url) == nil {
		hyperfleetURL = url
	}
}

// Reset clears the hyperfleet URL back to the zero value.
func Reset() { hyperfleetURL = "" }
