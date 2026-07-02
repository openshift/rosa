/*
Copyright (c) 2025 Red Hat, Inc.

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

package url

import (
	"fmt"
	neturl "net/url"
	"strings"
)

func ValidateURLCredentials(urlString string) error {
	parts := strings.SplitN(urlString, "://", 2)
	if len(parts) < 2 {
		return nil
	}
	rest := parts[1]

	if strings.Count(rest, "@") > 1 {
		return fmt.Errorf("password contains invalid character '@'")
	}

	userinfo, _, found := strings.Cut(rest, "@")
	if !found {
		return nil
	}

	username, password, hasPassword := strings.Cut(userinfo, ":")

	if err := checkForInvalidChars(username, "username"); err != nil {
		return err
	}

	if !hasPassword {
		return nil
	}

	return checkForInvalidChars(password, "password")
}

func checkForInvalidChars(value, field string) error {
	invalidChars := "/:#?[]@!$&'()*+,;="

	for _, char := range value {
		if strings.ContainsRune(invalidChars, char) {
			return fmt.Errorf("%s contains invalid character '%c'", field, char)
		}
	}

	return nil
}

// Parse mirrors net/url.Parse and rejects malformed IPv6 host literals early.
func Parse(rawURL string) (*neturl.URL, error) {
	return parse(rawURL, neturl.Parse)
}

// ParseRequestURI mirrors net/url.ParseRequestURI and rejects malformed IPv6 host literals early.
func ParseRequestURI(rawURL string) (*neturl.URL, error) {
	return parse(rawURL, neturl.ParseRequestURI)
}

func parse(rawURL string, parser func(string) (*neturl.URL, error)) (*neturl.URL, error) {
	if err := validateIPv6LiteralHost(rawURL); err != nil {
		return nil, err
	}

	return parser(rawURL)
}

func validateIPv6LiteralHost(rawURL string) error {
	authority, hasAuthority := getAuthority(rawURL)
	if !hasAuthority {
		return nil
	}

	if hostStart := strings.LastIndex(authority, "@"); hostStart != -1 {
		authority = authority[hostStart+1:]
	}

	if openBracketIdx := strings.LastIndex(authority, "["); openBracketIdx > 0 {
		return fmt.Errorf("invalid IP-literal")
	}

	return nil
}

func getAuthority(rawURL string) (string, bool) {
	_, after, ok := strings.Cut(rawURL, "://")
	if !ok {
		return "", false
	}

	authority := after
	if endIdx := strings.IndexAny(authority, "/?#"); endIdx != -1 {
		authority = authority[:endIdx]
	}

	return authority, true
}
