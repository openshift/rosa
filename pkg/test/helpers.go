package test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	accessv1 "github.com/openshift-online/ocm-sdk-go/accesstransparency/v1"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
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

func BuildBreakGlassCredential() *v1.BreakGlassCredential {
	const breakGlassCredentialId = "test-id"
	breakGlassCredential, err := v1.NewBreakGlassCredential().
		ID(breakGlassCredentialId).Username("username").Status(v1.BreakGlassCredentialStatusIssued).
		Build()
	Expect(err).To(BeNil())
	return breakGlassCredential
}

func BuildExternalAuth() *v1.ExternalAuth {
	const externalAuthName = "microsoft-entra-id"
	externalAuth, err := v1.NewExternalAuth().ID(externalAuthName).
		Issuer(v1.NewTokenIssuer().URL("https://test.com").Audiences("abc")).
		Claim(v1.NewExternalAuthClaim().Mappings(v1.NewTokenClaimMappings().
			UserName(v1.NewUsernameClaim().Claim("username")).
			Groups(v1.NewGroupsClaim().Claim("groups")))).
		Build()
	Expect(err).To(BeNil())
	return externalAuth
}

func MockNodePool(modifyFn func(n *v1.NodePoolBuilder)) *v1.NodePool {
	build := &v1.NodePoolBuilder{}
	if modifyFn != nil {
		modifyFn(build)
	}

	nodePool, err := build.Build()
	Expect(err).NotTo(HaveOccurred())
	return nodePool
}

func MockKubeletConfig(modifyFn func(k *v1.KubeletConfigBuilder)) *v1.KubeletConfig {
	build := &v1.KubeletConfigBuilder{}
	if modifyFn != nil {
		modifyFn(build)
	}

	kubeletConfig, err := build.Build()
	Expect(err).NotTo(HaveOccurred())
	return kubeletConfig
}

func MockAutoscaler(modifyFn func(a *v1.ClusterAutoscalerBuilder)) *v1.ClusterAutoscaler {
	build := &v1.ClusterAutoscalerBuilder{}
	if modifyFn != nil {
		modifyFn(build)
	}

	autoscaler, err := build.Build()
	Expect(err).NotTo(HaveOccurred())
	return autoscaler
}

func MockCluster(modifyFn func(c *v1.ClusterBuilder)) *v1.Cluster {
	mock := v1.NewCluster().
		ID(MockClusterID).
		HREF(MockClusterHREF).
		Name(MockClusterName)

	if modifyFn != nil {
		modifyFn(mock)
	}

	cluster, err := mock.Build()
	Expect(err).NotTo(HaveOccurred())
	return cluster
}

// FormatNodePoolAutoscaling simulates the output of APIs for a fake node pool list with autoscaling
func FormatNodePoolAutoscaling(nodePoolId string) string {
	version := v1.NewVersion().ID("4.12.24").RawID("openshift-4.12.24")
	awsNodePool := v1.NewAWSNodePool().InstanceType("m5.xlarge")
	nodeDrain := v1.NewValue().Value(1).Unit("minute")
	nodePool, err := v1.NewNodePool().ID(nodePoolId).Version(version).
		AWSNodePool(awsNodePool).AvailabilityZone("us-east-1a").NodeDrainGracePeriod(nodeDrain).
		Autoscaling(v1.NewNodePoolAutoscaling().ID("scaler").MinReplica(2).MaxReplica(1000)).Build()
	Expect(err).ToNot(HaveOccurred())
	return fmt.Sprintf("{\n  \"items\": [\n    %s\n  ],\n  \"page\": 0,\n  \"size\": 1,\n  \"total\": 1\n}",
		FormatResource(nodePool))
}

func FormatList[T any](list []*T, marshalFunc func([]*T, io.Writer) error, kind string) string {
	var json bytes.Buffer

	err := marshalFunc(list, &json)
	if err != nil {
		return err.Error()
	}

	return fmt.Sprintf(`{
        "kind": "%s",
        "page": 1,
        "size": %d,
        "total": %d,
        "items": %s
    }`, kind, len(list), len(list), json.String())
}

// Example usage for MachinePool and NodePool
func FormatMachinePoolList(machinePools []*v1.MachinePool) string {
	return FormatList(machinePools, v1.MarshalMachinePoolList, "MachinePoolList")
}

func FormatNodePoolList(nodePools []*v1.NodePool) string {
	return FormatList(nodePools, v1.MarshalNodePoolList, "NodePoolList")
}

func FormatKubeletConfigList(configs []*v1.KubeletConfig) string {
	return FormatList(configs, v1.MarshalKubeletConfigList, "KubeletConfigList")
}

func FormatClusterList(clusters []*v1.Cluster) string {
	return FormatList(clusters, v1.MarshalClusterList, "ClusterList")
}

func FormatIngressList(ingresses []*v1.Ingress) string {
	return FormatList(ingresses, v1.MarshalIngressList, "IngressList")
}

func FormatVersionList(versions []*v1.Version) string {
	return FormatList(versions, v1.MarshalVersionList, "VersionList")
}

func FormatMachineTypeList(mt []*v1.MachineType) string {
	return FormatList(mt, v1.MarshalMachineTypeList, "MachineTypeList")
}

func FormatTuningConfigList(tc []*v1.TuningConfig) string {
	return FormatList(tc, v1.MarshalTuningConfigList, "TuningConfigList")
}

func FormatQuotaCostList(qc []*amsv1.QuotaCost) string {
	return FormatList(qc, amsv1.MarshalQuotaCostList, "QuotaCostList")
}

func FormatIDPList(idps []*v1.IdentityProvider) string {
	return FormatList(idps, v1.MarshalIdentityProviderList, "IdentityProviderList")
}

func FormatHtpasswdUserList(htpasswdUsers []*v1.HTPasswdUser) string {
	return FormatList(htpasswdUsers, v1.MarshalHTPasswdUserList, "HTPasswdUserList")
}

func FormatExternalAuthList(externalAuths []*v1.ExternalAuth) string {
	return FormatList(externalAuths, v1.MarshalExternalAuthList, "ExternalAuthList")
}

func FormatNodePoolUpgradePolicyList(upgrades []*v1.NodePoolUpgradePolicy) string {
	return FormatList(upgrades, v1.MarshalNodePoolUpgradePolicyList, "NodePoolUpgradePolicyList")
}

func FormatAccessRequestList(accessRequests []*accessv1.AccessRequest) string {
	return FormatList(accessRequests, accessv1.MarshalAccessRequestList, "AccessRequestList")
}

func FormatAWSSTSPolicyList(upgrades []*v1.AWSSTSPolicy) string {
	return FormatList(upgrades, v1.MarshalAWSSTSPolicyList, "AWSSTSPolicyList")
}

// FormatResource wraps the SDK marshalling and returns a string starting from an object
func FormatResource(resource interface{}) string {
	var outputJson bytes.Buffer
	var err error
	switch reflect.TypeOf(resource).String() {
	case "*v1.KubeletConfig":
		if res, ok := resource.(*v1.KubeletConfig); ok {
			err = v1.MarshalKubeletConfig(res, &outputJson)
		}
	case "*v1.Version":
		if res, ok := resource.(*v1.Version); ok {
			err = v1.MarshalVersion(res, &outputJson)
		}
	case "*v1.NodePool":
		if res, ok := resource.(*v1.NodePool); ok {
			err = v1.MarshalNodePool(res, &outputJson)
		}
	case "*v1.MachinePool":
		if res, ok := resource.(*v1.MachinePool); ok {
			err = v1.MarshalMachinePool(res, &outputJson)
		}
	case "*v1.ClusterAutoscaler":
		if res, ok := resource.(*v1.ClusterAutoscaler); ok {
			err = v1.MarshalClusterAutoscaler(res, &outputJson)
		}
	case "*v1.ControlPlaneUpgradePolicy":
		if res, ok := resource.(*v1.ControlPlaneUpgradePolicy); ok {
			err = v1.MarshalControlPlaneUpgradePolicy(res, &outputJson)
		}
	case "*v1.ExternalAuth":
		if res, ok := resource.(*v1.ExternalAuth); ok {
			err = v1.MarshalExternalAuth(res, &outputJson)
		}
	case "*v1.BreakGlassCredential":
		if res, ok := resource.(*v1.BreakGlassCredential); ok {
			err = v1.MarshalBreakGlassCredential(res, &outputJson)
		}
	case "*v1.Account":
		if res, ok := resource.(*amsv1.Account); ok {
			err = amsv1.MarshalAccount(res, &outputJson)
		}
	case "*v1.Decision":
		if res, ok := resource.(*accessv1.Decision); ok {
			err = accessv1.MarshalDecision(res, &outputJson)
		}
	case "*v1.AccessRequest":
		if res, ok := resource.(*accessv1.AccessRequest); ok {
			err = accessv1.MarshalAccessRequest(res, &outputJson)
		}
	default:
		{
			return "NOTIMPLEMENTED"
		}
	}
	if err != nil {
		return err.Error()
	}

	return outputJson.String()
}

func NewTestRuntime() *TestingRuntime {
	t := &TestingRuntime{}
	t.InitRuntime()
	return t
}

// TestingRuntime is a wrapper for the structure used for testing
type TestingRuntime struct {
	SsoServer    *ghttp.Server
	ApiServer    *ghttp.Server
	RosaRuntime  *rosa.Runtime
	StdOutReader stdOutReader
}

func (t *TestingRuntime) InitRuntime() {
	// Create the servers:
	t.SsoServer = MakeTCPServer()
	t.ApiServer = MakeTCPServer()
	t.ApiServer.SetAllowUnhandledRequests(true)
	t.ApiServer.SetUnhandledRequestStatusCode(http.StatusInternalServerError)

	// Create the token:
	claims := MakeClaims()
	claims["username"] = "foo"
	accessTokenObj := MakeTokenObject(claims)
	accessToken := accessTokenObj.Raw

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

	ctrl := gomock.NewController(GinkgoT())
	aws := aws.NewMockClient(ctrl)
	t.RosaRuntime.AWSClient = aws

	DeferCleanup(t.RosaRuntime.Cleanup)
	DeferCleanup(t.SsoServer.Close)
	DeferCleanup(t.ApiServer.Close)
	DeferCleanup(t.Close)
}

func (t *TestingRuntime) Close() {
	ocm.SetClusterKey("")
}

func (t *TestingRuntime) SetCluster(clusterKey string, cluster *v1.Cluster) {
	ocm.SetClusterKey(clusterKey)
	t.RosaRuntime.Cluster = cluster
	t.RosaRuntime.ClusterKey = clusterKey
}

type stdOutReader struct {
	w           *os.File
	r           *os.File
	stdOutState *os.File
}

// Record pipes Stdout to a reader for returning all Stdout output with Read and saves the state of
// stdout to later return to normal. These two functions should be called in series
func (s *stdOutReader) Record() error {
	var err error
	s.stdOutState = os.Stdout
	s.r, s.w, err = os.Pipe()
	os.Stdout = s.w
	return err
}

// Read reads the output using the information gathered from Record, then returns Stdout to printing
// normally at the end of this function using the state captured from Record
func (s *stdOutReader) Read() (string, error) {
	err := s.w.Close()
	if err != nil {
		return "", err
	}
	out, err := io.ReadAll(s.r)
	os.Stdout = s.stdOutState

	return string(out), err
}
