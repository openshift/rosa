package aws_test

import (
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	. "github.com/onsi/ginkgo/v2"

	client "github.com/openshift/rosa/pkg/aws/api_interface"
	m "github.com/openshift/rosa/pkg/aws/mocks"
)

var _ = Describe("OrganizationsApiClient", func() {
	It("is implemented by AWS SDK Organizations Client", func() {
		awsOrganizationsClient := &organizations.Client{}
		var _ client.OrganizationsApiClient = awsOrganizationsClient
	})

	It("is implemented by MockOrganizationsApiClient", func() {
		mockOrganizationsApiClient := &m.MockOrganizationsApiClient{}
		var _ client.OrganizationsApiClient = mockOrganizationsApiClient
	})
})
