package aws_client

import "os"

func envCredential() bool {
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		return true
	}
	return false
}
func envAwsProfile() bool {
	return os.Getenv("AWS_SHARED_CREDENTIALS_FILE") != ""
}
