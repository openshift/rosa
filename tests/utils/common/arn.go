package common

import (
	"fmt"
	"net/url"
	"strings"
)

// Function to parse the ARN of a role, return rolePath,roleName and err
func ParseRoleARN(arn string) (string, string, error) {
	parts := strings.SplitN(arn, "role/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid ARN format")
	}
	u, err := url.Parse("/" + parts[1])
	if err != nil {
		return "", "", err
	}
	pathParts := strings.Split(u.Path, "/")
	roleName := pathParts[len(pathParts)-1]
	rolePath := strings.Join(pathParts[:len(pathParts)-1], "/")

	return rolePath, roleName, nil
}
