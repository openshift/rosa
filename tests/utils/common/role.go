package common

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// Extract aws commands to create AWS resource promted by rosacli, this function supports to parse bellow commands
// `rosa create account-role --mode manual`
// `rosa create operator-roles --mode manual`
// `rosa create oidc-provider --mode manual`
func ExtractCommandsToCreateAWSResoueces(bf bytes.Buffer) []string {
	var commands []string
	output := strings.Split(bf.String(), "\n\n")
	for _, message := range output {
		if strings.HasPrefix(message, "aws iam") {
			commands = append(commands, message)
		}
	}
	var newCommands []string
	for _, command := range commands {
		command = strings.ReplaceAll(command, "\\", "")
		command = strings.ReplaceAll(command, "\n", " ")
		spaceRegex := regexp.MustCompile(`\s+`)
		command = spaceRegex.ReplaceAllString(command, " ")
		command = strings.ReplaceAll(command, "'", "")
		newCommands = append(newCommands, command)

	}
	return newCommands
}

// Extract aws commands to create AWS resource promted by rosacli, this function supports to parse bellow commands
// `rosa create operator-roles --mode manual`
// `rosa create oidc-provider --mode manual`
func ExtractCommandsToDeleteAWSResoueces(bf bytes.Buffer) []string {
	var commands []string
	output := strings.Split(bf.String(), "\naws")
	for _, message := range output {
		if strings.HasPrefix(message, "aws iam") {
			commands = append(commands, message)
		} else {
			commands = append(commands, fmt.Sprintf("aws %s", message))
		}

	}
	var newCommands []string
	for _, command := range commands {
		command = strings.ReplaceAll(command, "\\", "")
		command = strings.ReplaceAll(command, "\n", " ")
		spaceRegex := regexp.MustCompile(`\s+`)
		command = spaceRegex.ReplaceAllString(command, " ")
		command = strings.ReplaceAll(command, "'", "")
		newCommands = append(newCommands, command)

	}
	return newCommands
}
