package aws_test

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	. "github.com/onsi/ginkgo/v2"

	client "github.com/openshift/rosa/pkg/aws/api_interface"
	m "github.com/openshift/rosa/pkg/aws/mocks"
)

var _ = Describe("CloudFormationApiClient", func() {
	It("is implemented by AWS SDK CloudFormation Client", func() {
		awsCloudFormationClient := &cloudformation.Client{}
		var _ client.CloudFormationApiClient = awsCloudFormationClient
	})

	It("is implemented by MockCloudFormationApiClient", func() {
		mockCloudFormationApiClient := &m.MockCloudFormationApiClient{}
		var _ client.CloudFormationApiClient = mockCloudFormationApiClient
	})
})
