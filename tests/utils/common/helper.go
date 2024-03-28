package common

import "os"

func ReadENVWithDefaultValue(envName string, fallback string) string {
	if os.Getenv(envName) != "" {
		return os.Getenv(envName)
	}
	return fallback
}
