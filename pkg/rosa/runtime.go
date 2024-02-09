package rosa

import (
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/reporter"
)

type Runtime struct {
	Reporter   *reporter.Object
	Logger     *logrus.Logger
	OCMClient  *ocm.Client
	AWSClient  aws.Client
	Creator    *aws.Creator
	ClusterKey string
	Cluster    *cmv1.Cluster
}

func NewRuntime() *Runtime {
	reporter := reporter.CreateReporterOrExit()
	logger := logging.NewLogger()
	return &Runtime{Reporter: reporter, Logger: logger}
}

// Adds an OCM client to the runtime. Requires a deferred call to `.Cleanup()` to close connections.
func (r *Runtime) WithOCM() *Runtime {
	if r.OCMClient == nil {
		r.OCMClient = ocm.CreateNewClientOrExit(r.Logger, r.Reporter)
	}
	return r
}

// Adds an AWS client to the runtime
func (r *Runtime) WithAWS() *Runtime {
	if r.AWSClient == nil {
		r.AWSClient = aws.CreateNewClientOrExit(r.Logger, r.Reporter)
	}
	if r.Creator == nil {
		var err error
		r.Creator, err = r.AWSClient.GetCreator()
		if err != nil {
			r.Reporter.Errorf("Failed to get AWS creator: %v", err)
			os.Exit(1)
		}
	}
	return r
}

func (r *Runtime) Cleanup() {
	if r.OCMClient != nil {
		if err := r.OCMClient.Close(); err != nil {
			r.Reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}
}

// Load the cluster key provided by the user into the runtime and return it
func (r *Runtime) GetClusterKey() string {
	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	r.ClusterKey = clusterKey
	return clusterKey
}

func (r *Runtime) FetchCluster() *cmv1.Cluster {
	if r.Cluster != nil {
		return r.Cluster
	}

	// We don't want to lazy init the OCM client since it requires cleanup
	if r.OCMClient == nil {
		r.Reporter.Errorf("Tried to fetch a cluster without initializing the OCM client, exiting.")
		os.Exit(1)
	}
	if r.ClusterKey == "" {
		r.GetClusterKey()
	}
	if r.Creator == nil {
		r.WithAWS()
	}

	r.Reporter.Debugf("Loading cluster '%s'", r.ClusterKey)
	cluster, err := r.OCMClient.GetCluster(r.ClusterKey, r.Creator)
	if err != nil {
		r.Reporter.Errorf("Failed to get cluster '%s': %v", r.ClusterKey, err)
		os.Exit(1)
	}
	r.Cluster = cluster
	return cluster
}
