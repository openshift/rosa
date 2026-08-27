package hyperfleet

import "github.com/spf13/cobra"

var (
	hyperfleetURL string
	urlFromFlag   bool
)

type urlFlag struct{}

func (urlFlag) String() string { return hyperfleetURL }
func (urlFlag) Type() string   { return "string" }
func (urlFlag) Set(v string) error {
	hyperfleetURL = v
	urlFromFlag = true
	return nil
}

func AddFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().Var(
		urlFlag{},
		"hyperfleet-url",
		"Platform API v2 endpoint URL. When set, commands route to the Platform API instead of OCM.",
	)
	_ = cmd.PersistentFlags().MarkHidden("hyperfleet-url")
}

func Enabled() bool       { return hyperfleetURL != "" }
func ExplicitURL() string { return hyperfleetURL }

// FromFlag reports whether --hyperfleet-url was passed on the command line,
// as opposed to seeded from config via SetURL.
func FromFlag() bool { return urlFromFlag }

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
func Reset() {
	hyperfleetURL = ""
	urlFromFlag = false
}
