package aws_test

import (
	"github.com/aws/aws-sdk-go-v2/service/iam"
	. "github.com/onsi/ginkgo/v2"

	client "github.com/openshift/rosa/pkg/aws/api_interface"
	m "github.com/openshift/rosa/pkg/aws/mocks"
)

var _ = Describe("IamApiClient", func() {
	It("is implemented by AWS SDK IAM Client", func() {
		awsIamClient := &iam.Client{}
		var _ client.IamApiClient = awsIamClient
	})

	It("is implemented by MockIamApiClient", func() {
		mockIamApiClient := &m.MockIamApiClient{}
		var _ client.IamApiClient = mockIamApiClient
	})
})
