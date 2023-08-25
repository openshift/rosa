package test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
)

func RunWithOutputCapture(runWithRuntime func(*rosa.Runtime, *cobra.Command) error,
	runtime *rosa.Runtime, cmd *cobra.Command) (string, string, error) {
	var err error
	var stdout []byte
	var stderr []byte

	rout, wout, _ := os.Pipe()
	tmpout := os.Stdout
	rerr, werr, _ := os.Pipe()
	tmperr := os.Stderr
	defer func() {
		os.Stdout = tmpout
		os.Stderr = tmperr
	}()
	os.Stdout = wout
	os.Stderr = werr

	go func() {
		err = runWithRuntime(runtime, cmd)
		wout.Close()
		werr.Close()
	}()
	stdout, _ = io.ReadAll(rout)
	stderr, _ = io.ReadAll(rerr)

	return string(stdout), string(stderr), err
}

func RunWithOutputCaptureAndArgv(runWithRuntime func(*rosa.Runtime, *cobra.Command, []string) error,
	runtime *rosa.Runtime, cmd *cobra.Command, argv *[]string) (string, string, error) {
	var err error
	var stdout []byte
	var stderr []byte

	rout, wout, _ := os.Pipe()
	tmpout := os.Stdout
	rerr, werr, _ := os.Pipe()
	tmperr := os.Stderr
	defer func() {
		os.Stdout = tmpout
		os.Stderr = tmperr
	}()
	os.Stdout = wout
	os.Stderr = werr

	go func() {
		err = runWithRuntime(runtime, cmd, *argv)
		wout.Close()
		werr.Close()
	}()
	stdout, _ = io.ReadAll(rout)
	stderr, _ = io.ReadAll(rerr)

	return string(stdout), string(stderr), err
}

var (
	MockClusterID   = "24vf9iitg3p6tlml88iml6j6mu095mh8"
	MockClusterHREF = "/api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8"
	MockClusterName = "cluster"
)

func MockOCMCluster(modifyFn func(c *v1.ClusterBuilder)) (*v1.Cluster, error) {
	mock := v1.NewCluster().
		ID(MockClusterID).
		HREF(MockClusterHREF).
		Name(MockClusterName)

	if modifyFn != nil {
		modifyFn(mock)
	}

	return mock.Build()
}

func FormatClusterList(clusters []*v1.Cluster) string {
	var clusterJson bytes.Buffer

	v1.MarshalClusterList(clusters, &clusterJson)

	return fmt.Sprintf(`
	{
		"kind": "ClusterList",
		"page": 1,
		"size": %d,
		"total": %d,
		"items": %s
	}`, len(clusters), len(clusters), clusterJson.String())
}

func FormatIngressList(ingresses []*v1.Ingress) string {
	var ingressJson bytes.Buffer

	v1.MarshalIngressList(ingresses, &ingressJson)

	return fmt.Sprintf(`
	{
		"kind": "IngressList",
		"page": 1,
		"size": %d,
		"total": %d,
		"items": %s
	}`, len(ingresses), len(ingresses), ingressJson.String())
}

func FormatNodePoolUpgradePolicyList(upgrades []*v1.NodePoolUpgradePolicy) string {
	var outputJson bytes.Buffer

	v1.MarshalNodePoolUpgradePolicyList(upgrades, &outputJson)

	return fmt.Sprintf(`
	{
		"kind": "NodePoolUpgradePolicyList",
		"page": 1,
		"size": %d,
		"total": %d,
		"items": %s
	}`, len(upgrades), len(upgrades), outputJson.String())
}

// TestingRuntime is a wrapper for the structure used for testing
type TestingRuntime struct {
	SsoServer   *ghttp.Server
	ApiServer   *ghttp.Server
	RosaRuntime *rosa.Runtime
}

func (t *TestingRuntime) InitRuntime() {
	// Create the servers:
	t.SsoServer = MakeTCPServer()
	t.ApiServer = MakeTCPServer()
	t.ApiServer.SetAllowUnhandledRequests(true)
	t.ApiServer.SetUnhandledRequestStatusCode(http.StatusInternalServerError)

	// Create the token:
	accessToken := MakeTokenString("Bearer", 15*time.Minute)

	// Prepare the server:
	t.SsoServer.AppendHandlers(
		RespondWithAccessToken(accessToken),
	)
	// Prepare the logger:
	logger, err := logging.NewGoLoggerBuilder().
		Debug(true).
		Build()
	Expect(err).To(BeNil())
	// Set up the connection with the fake config
	connection, err := sdk.NewConnectionBuilder().
		Logger(logger).
		Tokens(accessToken).
		URL(t.ApiServer.URL()).
		Build()
	// Initialize client object
	Expect(err).To(BeNil())
	ocmClient := ocm.NewClientWithConnection(connection)
	ocm.SetClusterKey("cluster1")
	t.RosaRuntime = rosa.NewRuntime()
	t.RosaRuntime.OCMClient = ocmClient
	t.RosaRuntime.Creator = &aws.Creator{
		ARN:       "fake",
		AccountID: "123",
		IsSTS:     false,
	}
	DeferCleanup(t.RosaRuntime.Cleanup)
	DeferCleanup(t.SsoServer.Close)
	DeferCleanup(t.ApiServer.Close)
}
