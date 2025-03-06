package helper

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	. "github.com/openshift/rosa/tests/utils/log"
)

// Split resources from the aws arn
func SplitARNResources(v string) []string {
	var parts []string
	var offset int

	for offset <= len(v) {
		idx := strings.IndexAny(v[offset:], "/:")
		if idx < 0 {
			parts = append(parts, v[offset:])
			break
		}
		parts = append(parts, v[offset:idx+offset])
		offset += idx + 1
	}
	return parts
}

// Extract the oidc provider ARN from the output of `rosa create oidc-config --mode auto`
// and also for common message containing the arn
func ExtractOIDCProviderARN(output string) string {
	oidcProviderArnRE := regexp.MustCompile(`arn:aws:iam::[^']+:oidc-provider/[^']+`)
	submatchall := oidcProviderArnRE.FindAllString(output, -1)
	if len(submatchall) < 1 {
		Logger.Warnf("Cannot find sub string matached %s from input string %s! Please check the matching string",
			oidcProviderArnRE,
			output)
		return ""
	}
	if len(submatchall) > 1 {
		Logger.Warnf("Find more than one sub string matached %s! "+
			"Please check this unexpexted result then update the regex if needed.",
			oidcProviderArnRE)
	}
	return submatchall[0]
}

// Extract the oidc provider ARN from the output of `rosa create oidc-config --mode auto`
// and also for common message containing the arn
func ExtractOIDCProviderIDFromARN(arn string) string {
	spliptElements := SplitARNResources(arn)
	return spliptElements[len(spliptElements)-1]
}

func ExtractCommandsFromOIDCRegister(bf bytes.Buffer) []string {
	var commands []string
	commands = strings.Split(bf.String(), "\n\n")
	// When the rosacli is not the latest version, there will be warning message
	// This will remove the warning message from the topest command
	if strings.Contains(commands[0], "WARN") && strings.Contains(commands[0], "aws") {
		splitCommands := strings.Split(commands[0], "\naws")
		commands[0] = fmt.Sprintf("aws %s", splitCommands[1])
	}
	for k, command := range commands {
		if strings.Contains(command, "\naws") {
			splitCommands := strings.Split(command, "\naws")
			commands[k] = splitCommands[0]
			commands = append(commands, fmt.Sprintf("aws %s", splitCommands[1]))
		}
	}

	var newCommands []string
	for _, command := range commands {
		if strings.Contains(command, "WARN") {
			continue
		}
		command = strings.ReplaceAll(command, "\\", "")
		command = strings.ReplaceAll(command, "\n", " ")
		spaceRegex := regexp.MustCompile(`\s+`)
		command = spaceRegex.ReplaceAllString(command, " ")
		// remove '' in the value
		command = strings.ReplaceAll(command, "'", "")
		if strings.Contains(command, " INFO") {
			command = strings.Split(command, " INFO")[0]
		}
		newCommands = append(newCommands, command)

	}
	return newCommands
}

// Parse command string to args array. NOTE:If the flag value contains spaces, put the whole value into the array
func ParseCommandToArgs(command string) []string {
	var args []string
	re := regexp.MustCompile(`'[^']*'|"[^"]*"|\S+`)
	matches := re.FindAllString(command, -1)

	for _, match := range matches {
		cleanedMatch := strings.Trim(match, `"'`)
		args = append(args, cleanedMatch)
	}
	return args
}

func ParseSecretArnFromOutput(output string) string {
	re := regexp.MustCompile(`"ARN":\s*"([^"]+)"`)
	matches := re.FindStringSubmatch(output)

	if len(matches) > 1 {
		return matches[1]
	} else {
		Logger.Warnf("sercret manager ARN not found in %s", output)
		return ""
	}
}

func ParseIssuerURLFromCommand(command string) string {
	re := regexp.MustCompile(`https://[^\s]+`)
	return re.FindString(command)
}

// Extract oidc provider from the 'OIDC Endpoint URL' field of `rosa describe cluster`
func ExtractOIDCProviderFromOidcUrl(urlString string) (string, error) {
	var oidcProvider string
	urlString = strings.TrimSpace(strings.Split(urlString, " ")[0])
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return "", err
	}
	host := parsedURL.Host
	path := strings.TrimPrefix(parsedURL.Path, "/")
	oidcProvider = host
	if path != "" {
		oidcProvider = fmt.Sprintf("%s/%s", host, path)
	}
	return oidcProvider, nil
}
