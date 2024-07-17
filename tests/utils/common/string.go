package common

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
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

func ParseTagsFronJsonOutput(tags string) map[string]interface{} {
	output := make(map[string]interface{})
	rawMap := tags[strings.Index(tags, "map[")+4 : strings.LastIndex(tags, "]")]
	pairs := strings.Split(rawMap, " ")
	for _, pair := range pairs {
		kvp := strings.SplitN(pair, ":", 2)
		if len(kvp) != 2 {
			fmt.Printf("Bad key-value pair, ignoring...")
			continue
		}
		output[kvp[0]] = kvp[1]
	}
	return output
}

func GenerateRandomStringWithSymbols(length int) string {
	b := make([]byte, length)
	_, err := rand.Read(b)
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
func GenerateRandomString(length int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	ret := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			panic(err)
		}
		ret[i] = letters[num.Int64()]
	}

	return string(ret)
}

func GenerateRandomName(prefix string, n int) string {
	return fmt.Sprintf("%s-%s", prefix, strings.ToLower(GenerateRandomString(n)))
}

func TrimNameByLength(name string, length int) string {
	if len(name) <= length {
		return name
	}
	return strings.TrimSuffix(name[0:length], "-")
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
func ReplaceCommaSpaceWithComma(sourceValue string) string {
	splited := ParseCommaSeparatedStrings(sourceValue)
	return strings.Join(splited, ",")
}
