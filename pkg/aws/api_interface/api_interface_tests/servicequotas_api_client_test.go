package aws_test

import (
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	. "github.com/onsi/ginkgo/v2"

	client "github.com/openshift/rosa/pkg/aws/api_interface"
	m "github.com/openshift/rosa/pkg/aws/mocks"
)

var _ = Describe("ServiceQuotasApiClient", func() {
	It("is implemented by AWS SDK Service Quotas Client", func() {
		awsServiceQuotasClient := &servicequotas.Client{}
		var _ client.ServiceQuotasApiClient = awsServiceQuotasClient
	})

	It("is implemented by MockServiceQuotasApiClient", func() {
		mockServiceQuotasApiClient := &m.MockServiceQuotasApiClient{}
		var _ client.ServiceQuotasApiClient = mockServiceQuotasApiClient
	})
})
