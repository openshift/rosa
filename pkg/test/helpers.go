package test

import (
	"bytes"
	"fmt"
	"io"
	"os"

	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
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
