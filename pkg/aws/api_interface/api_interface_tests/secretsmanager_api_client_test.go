package aws_test

import (
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	. "github.com/onsi/ginkgo/v2"

	client "github.com/openshift/rosa/pkg/aws/api_interface"
	m "github.com/openshift/rosa/pkg/aws/mocks"
)

var _ = Describe("SecretsManagerApiClient", func() {
	It("is implemented by AWS SDK Secrets Manager Client", func() {
		awsSecretsManagerClient := &secretsmanager.Client{}
		var _ client.SecretsManagerApiClient = awsSecretsManagerClient
	})

	It("is implemented by MockSecretsManagerApiClient", func() {
		mockSecretsManagerApiClient := &m.MockSecretsManagerApiClient{}
		var _ client.SecretsManagerApiClient = mockSecretsManagerApiClient
	})
})
