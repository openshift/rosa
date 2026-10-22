// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package seaplane

var (
	seaplaneURL string
	urlFromFlag bool
)

// FlagValue is a pflag.Value for --seaplane-url. Register it from cmd/, not here:
// pkg/seaplane is core and cannot import cobra.
type FlagValue struct{}

func (FlagValue) String() string { return seaplaneURL }
func (FlagValue) Type() string   { return "string" }
func (FlagValue) Set(v string) error {
	seaplaneURL = v
	urlFromFlag = true
	return nil
}

func Enabled() bool       { return seaplaneURL != "" }
func ExplicitURL() string { return seaplaneURL }

// FromFlag reports whether --seaplane-url was passed on the command line,
// as opposed to seeded from config via SetURL.
func FromFlag() bool { return urlFromFlag }

// SetFromFlag records a command-line --seaplane-url (tests and flag parsing).
func SetFromFlag(url string) {
	_ = FlagValue{}.Set(url)
}

// SetURL seeds the seaplane URL from an external source (e.g. stored config)
// when the --seaplane-url flag was not passed explicitly. A flag value always
// takes precedence: if Enabled() is already true, this is a no-op. Non-HTTPS
// URLs are silently rejected to prevent cleartext endpoints from being used.
func SetURL(url string) {
	if seaplaneURL == "" && ValidateURL(url) == nil {
		seaplaneURL = url
	}
}

// Reset clears the seaplane URL back to the zero value.
func Reset() {
	seaplaneURL = ""
	urlFromFlag = false
}
