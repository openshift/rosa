package test

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	hfclient "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	hfrest "github.com/openshift-online/rosa-hyperfleet-api/clientset/rest"

	"github.com/openshift/rosa/pkg/rosa"
)

// NewHyperfleetTestRuntime provides the real SigV4 client with local HTTP handlers,
// without initializing an OCM connection or reading local AWS credentials.
func NewHyperfleetTestRuntime() (*rosa.Runtime, *ghttp.Server) {
	server := ghttp.NewServer()
	ginkgo.DeferCleanup(server.Close)
	client, err := hfclient.NewForConfig(&hfrest.Config{
		Host: server.URL(), Region: "us-east-1", AccountID: "123456789012",
		AWSConfig: aws.Config{Credentials: credentials.NewStaticCredentialsProvider("test", "test", "")},
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	r := rosa.NewRuntime()
	r.HyperFleetClient = client
	return r, server
}
