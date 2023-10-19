package validations

import (
	"fmt"
	"regexp"
	"strings"
)

func PasswordValidator(val interface{}) error {
	if password, ok := val.(string); ok {
		re := regexp.MustCompile(`[^\x20-\x7E]`)
		invalidChars := re.FindAllString(password, -1)
		notAsciiOnly := len(invalidChars) > 0
		containsSpace := strings.Contains(password, " ")
		tooShort := len(password) < 14
		pwdErrors := []string{}
		if notAsciiOnly {
			pwdErrors = append(pwdErrors, fmt.Sprintf("must not contain special characters [%s]",
				strings.Join(invalidChars, ", ")))
		}
		if containsSpace {
			pwdErrors = append(pwdErrors, "must not contain whitespace")
		}
		if tooShort {
			pwdErrors = append(pwdErrors, fmt.Sprintf("must be at least 14 characters (got %d)", len(password)))
		}
		if notAsciiOnly || containsSpace || tooShort {
			if len(pwdErrors) > 1 {
				pwdErrors[len(pwdErrors)-1] = "and " + pwdErrors[len(pwdErrors)-1]
			}

			return fmt.Errorf("Password " + strings.Join(pwdErrors, ", "))
		}
		hasUppercase, _ := regexp.MatchString(`[A-Z]`, password)
		hasLowercase, _ := regexp.MatchString(`[a-z]`, password)
		hasNumberOrSymbol, _ := regexp.MatchString(`[^a-zA-Z]`, password)
		if !hasUppercase || !hasLowercase || !hasNumberOrSymbol {
			return fmt.Errorf(
				"Password must include uppercase letters, lowercase letters, and numbers " +
					"or symbols (ASCII-standard characters only)")
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got '%v'", val)
}