package aws_test

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"
	. "github.com/onsi/ginkgo/v2"
	f "github.com/openshift/rosa/pkg/aws/api_interface"
	m "github.com/openshift/rosa/pkg/aws/mocks"
)

var _ = Describe("S3ApiClient", func() {
	It("is implemented by AWS SDK S3 Client", func() {
		awsS3Client := &s3.Client{}
		var _ f.S3ApiClient = awsS3Client
	})

	It("is implemented by MockS3ApiClient", func() {
		mockS3ApiClient := &m.MockS3ApiClient{}
		var _ f.S3ApiClient = mockS3ApiClient
	})
})
