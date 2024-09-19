package helper

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	. "github.com/openshift/rosa/tests/utils/log"
)

// Generate htpasspwd key value pair, return with a string
func GenerateHtpasswdPair(user string, pass string) (string, string, string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		Logger.Errorf("Fail to generate htpasswd file: %v", err)
		return "", "", "", err
	}
	htpasswdPair := fmt.Sprintf("%s:%s", user, string(hashedPassword))
	parts := strings.SplitN(htpasswdPair, ":", 2)
	return htpasswdPair, parts[0], parts[1], nil
}

// generate Htpasswd user-password Pairs
func GenerateMultipleHtpasswdPairs(pairNum int) ([]string, error) {
	multipleuserPasswd := []string{}
	for i := 0; i < pairNum; i++ {
		userPasswdPair, _, _, err := GenerateHtpasswdPair(GenerateRandomString(6), GenerateRandomString(6))
		if err != nil {
			return multipleuserPasswd, err
		}
		multipleuserPasswd = append(multipleuserPasswd, userPasswdPair)
	}
	return multipleuserPasswd, nil
}
