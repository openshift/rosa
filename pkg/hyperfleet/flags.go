package hyperfleet

var (
	hyperfleetURL string
	urlFromFlag   bool
)

// FlagValue is a pflag.Value for --hyperfleet-url. Register it from cmd/, not here:
// pkg/hyperfleet is core and cannot import cobra.
type FlagValue struct{}

func (FlagValue) String() string { return hyperfleetURL }
func (FlagValue) Type() string   { return "string" }
func (FlagValue) Set(v string) error {
	hyperfleetURL = v
	urlFromFlag = true
	return nil
}

func Enabled() bool       { return hyperfleetURL != "" }
func ExplicitURL() string { return hyperfleetURL }

// FromFlag reports whether --hyperfleet-url was passed on the command line,
// as opposed to seeded from config via SetURL.
func FromFlag() bool { return urlFromFlag }

// SetFromFlag records a command-line --hyperfleet-url (tests and flag parsing).
func SetFromFlag(url string) {
	_ = FlagValue{}.Set(url)
}

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
