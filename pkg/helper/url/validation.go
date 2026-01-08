package url

import "strings"

// URLCredentialValidation contains the result of validating URL credentials
type URLCredentialValidation struct {
	Field       string
	InvalidChar rune
	Error       string
}

// ValidateURLCredentials checks for invalid characters in URL username and password
func ValidateURLCredentials(urlString string) *URLCredentialValidation {
	if !strings.Contains(urlString, "://") {
		return &URLCredentialValidation{Error: "URL is missing scheme (expected '://')"}
	}
	if !strings.Contains(urlString, "@") {
		return &URLCredentialValidation{}
	}

	schemeIdx := strings.Index(urlString, "://")
	rest := urlString[schemeIdx+3:]
	atIdx := strings.LastIndex(rest, "@")
	if atIdx == -1 {
		return &URLCredentialValidation{}
	}

	userinfo := rest[:atIdx]

	if strings.Count(urlString, "@") > 1 {
		return &URLCredentialValidation{
			Field:       "password",
			InvalidChar: '@',
		}
	}

	colonIdx := strings.Index(userinfo, ":")
	if colonIdx == -1 {
		return checkForInvalidChars(userinfo, "username")
	}

	username := userinfo[:colonIdx]
	if result := checkForInvalidChars(username, "username"); result.InvalidChar != 0 {
		return result
	}

	password := userinfo[colonIdx+1:]
	return checkForInvalidChars(password, "password")
}

func checkForInvalidChars(value, field string) *URLCredentialValidation {
	invalidChars := "/:#?[]@!$&'()*+,;="

	for _, char := range value {
		if strings.ContainsRune(invalidChars, char) {
			return &URLCredentialValidation{
				Field:       field,
				InvalidChar: char,
			}
		}
	}

	return &URLCredentialValidation{}
}
