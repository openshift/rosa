package common

import (
	r "crypto/rand"
	"encoding/base64"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

func ParseLabels(labels string) []string {
	return ParseCommaSeparatedStrings(labels)
}

func ParseTaints(taints string) []string {
	return ParseCommaSeparatedStrings(taints)
}

func ParseTuningConfigs(tuningConfigs string) []string {
	return ParseCommaSeparatedStrings(tuningConfigs)
}

func ParseCommaSeparatedStrings(input string) (output []string) {
	split := strings.Split(strings.ReplaceAll(input, " ", ""), ",")
	for _, item := range split {
		if strings.TrimSpace(item) != "" {
			output = append(output, item)
		}
	}
	return
}
func GenerateRandomStringWithSymbols(length int) string {
	b := make([]byte, length)
	_, err := r.Read(b)
	if err != nil {
		panic(err)
	}
	randomString := base64.StdEncoding.EncodeToString(b)[:length]
	f := func(r rune) bool {
		return r < 'A' || r > 'z'
	}
	// Verify that the string contains special character or number
	if strings.IndexFunc(randomString, f) == -1 {
		randomString = randomString[:len(randomString)-1] + "!"
	}
	return randomString
}

// Generate random string
func GenerateRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())

	s := make([]byte, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func GenerateRandomName(prefix string, n int) string {
	return fmt.Sprintf("%s-%s", prefix, strings.ToLower(GenerateRandomString(n)))
}

func TrimNameByLength(name string, length int) string {
	if len(name) <= length {
		return name
	}
	return name[0:length]
}

func SplitMajorVersion(openshiftVersion string) string {
	splited := strings.Split(openshiftVersion, ".")
	if len(splited) < 2 {
		return openshiftVersion
	}
	return strings.Join(splited[0:2], ".")
}

func ReplaceCommaWithCommaSpace(sourceValue string) string {
	splited := ParseCommaSeparatedStrings(sourceValue)
	return strings.Join(splited, ", ")
}
