package url

import (
	"fmt"
	"strings"
)

// ValidateURLCredentials checks for invalid characters in URL username and password
func ValidateURLCredentials(urlString string) error {
	if !strings.Contains(urlString, "://") {
		return fmt.Errorf("URL is missing scheme (expected '://')")
	}

	schemeIdx := strings.Index(urlString, "://")
	rest := urlString[schemeIdx+3:]
	atIdx := strings.LastIndex(rest, "@")
	if atIdx == -1 {
		return nil
	}

	userinfo := rest[:atIdx]

	if strings.Count(urlString, "@") > 1 {
		return fmt.Errorf("password contains invalid character '@'")
	}

	colonIdx := strings.Index(userinfo, ":")

	var username string
	if colonIdx == -1 {
		username = userinfo
	} else {
		username = userinfo[:colonIdx]
	}

	if err := checkForInvalidChars(username, "username"); err != nil {
		return err
	}

	if colonIdx == -1 {
		return nil
	}

	password := userinfo[colonIdx+1:]
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
