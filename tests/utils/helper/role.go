package helper

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
func ExtractCommandsToCreateAWSResources(bf bytes.Buffer) []string {
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
		// convert json string of --policy-document which account roles
		// of shared-vpc hosted-cp cluster
		if strings.Contains(command, "--policy-document {") {
			start := strings.Index(command, "--policy-document {") + len("--policy-document ")
			end := strings.Index(command, "--policy-name") - 1
			jsonString := command[start:end]
			jsonString = strings.ReplaceAll(jsonString, " ", "")

			command = command[:start] + jsonString + command[end:]
		}
		newCommands = append(newCommands, command)

	}
	return newCommands
}

// Extract aws commands to delete AWS resource promted by rosacli, this function supports to parse bellow commands
// `rosa delete operator-roles --mode manual`
// `rosa delete oidc-provider --mode manual`
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

// Extract aws command to delete account roles in manual mode
func ExtractCommandsToDeleteAccountRoles(bf bytes.Buffer) []string {
	var commands []string
	output := strings.Split(bf.String(), "\n\n")
	for _, message := range output {
		if strings.HasPrefix(message, "aws iam") {
			commands = append(commands, message)
		}
	}
	var newCommands []string
	for _, command := range commands {
		if strings.Contains(command, "WARN") {
			awscommand := strings.Split(command, "\nWARN")
			command = awscommand[0]
		}
		command = strings.ReplaceAll(command, "\\", "")
		command = strings.ReplaceAll(command, "\n", " ")
		spaceRegex := regexp.MustCompile(`\s+`)
		command = spaceRegex.ReplaceAllString(command, " ")
		command = strings.ReplaceAll(command, "'", "")
		newCommands = append(newCommands, command)

	}
	return newCommands
}

// Extract aws commands to create AWS resource promted by rosacli, this function supports to parse below commands
// `rosa create cluster --mode manual --sts /--hosted-cp`
func ExtractAWSCmdsForClusterCreation(bf bytes.Buffer) []string {
	var commands []string
	//remove empty lines
	lines := strings.Split(bf.String(), "\n")
	var nonEmptyLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}
	nonEmptyInput := strings.Join(nonEmptyLines, "\n")
	//replace \ and \n and spaces with one space
	re := regexp.MustCompile(`\\\s*\n\s*`)
	processedInput := re.ReplaceAllString(nonEmptyInput, " ")

	output := strings.Split(processedInput, "\n")
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
