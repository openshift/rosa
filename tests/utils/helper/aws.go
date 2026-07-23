package helper

import (
	"fmt"
	"strings"

	//nolint:staticcheck
	. "github.com/onsi/ginkgo/v2"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"
)

// credentialErrorSubstrings match STS/AWS auth failures seen in CI when tokens expire mid-run.
var credentialErrorSubstrings = []string{
	"InvalidClientTokenId",
	"ExpiredToken",
	"RequestExpired",
	"security token included in the request is invalid",
	"security token included in the request is expired",
	"AWS was not able to validate the provided access credentials",
	"NoCredentialProviders",
	"failed to refresh cached credentials",
}

// createAWSClientForCredCheck is overridable in unit tests.
var createAWSClientForCredCheck = func() (*aws_client.AWSClient, error) {
	return aws_client.CreateAWSClient("", "")
}

// skipForInfra is overridable in unit tests (defaults to Ginkgo Skip).
var skipForInfra = func(message string, callerSkip ...int) {
	Skip(message, callerSkip...)
}

// documentStep is overridable in unit tests (defaults to Ginkgo By).
var documentStep = By

// AWSCredentialsInvalid reports whether a cheap STS check shows expired/invalid AWS credentials.
// invalid=false means credentials look usable (or the failure is not a recognized credential error).
func AWSCredentialsInvalid() (invalid bool, err error) {
	_, err = createAWSClientForCredCheck()
	if err == nil {
		return false, nil
	}
	if isAWSCredentialError(err.Error()) {
		return true, err
	}
	return false, err
}

// SkipIfAWSCredentialsInvalid performs a cheap STS GetCallerIdentity via CreateAWSClient.
// On expired/invalid credentials it Skips so Ginkgo records infra noise instead of a product failure.
func SkipIfAWSCredentialsInvalid() {
	GinkgoHelper()
	documentStep("Check AWS credentials are valid")
	invalid, err := AWSCredentialsInvalid()
	if invalid {
		skipForInfra(fmt.Sprintf("AWS credentials invalid/expired — infra, not product: %v", err))
	}
}

func isAWSCredentialError(errMsg string) bool {
	lower := strings.ToLower(errMsg)
	for _, substr := range credentialErrorSubstrings {
		if strings.Contains(lower, strings.ToLower(substr)) {
			return true
		}
	}
	return false
}
