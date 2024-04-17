package aws_test

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	. "github.com/onsi/ginkgo/v2"

	client "github.com/openshift/rosa/pkg/aws/api_interface"
	m "github.com/openshift/rosa/pkg/aws/mocks"
)

var _ = Describe("Ec2ApiClient", func() {
	It("is implemented by AWS SDK EC2 Client", func() {
		awsEC2Client := &ec2.Client{}
		var _ client.Ec2ApiClient = awsEC2Client
	})

	It("is implemented by MockEc2ApiClient", func() {
		mockEc2ApiClient := &m.MockEc2ApiClient{}
		var _ client.Ec2ApiClient = mockEc2ApiClient
	})
})
