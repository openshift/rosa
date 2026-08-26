package rosa

import (
	"context"
	"os"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awssts "github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/briandowns/spinner"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	hfrest "github.com/openshift-online/rosa-hyperfleet-api/clientset/rest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/profile"
	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/reporter"
)

// awsLoadConfig, awsGetIdentity, hfNewClient, hfExplicitURL, hfExitFn are
// package-level vars so they can be replaced in unit tests without interface indirection.
var (
	awsLoadConfig = func(ctx context.Context, region, awsProfile string) (awssdk.Config, error) {
		return awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(region),
			awsconfig.WithSharedConfigProfile(awsProfile),
		)
	}
	awsGetIdentity = func(ctx context.Context, cfg awssdk.Config) (*awssts.GetCallerIdentityOutput, error) {
		return awssts.NewFromConfig(cfg).GetCallerIdentity(ctx, &awssts.GetCallerIdentityInput{})
	}
	hfNewClient   = hyperfleetclientset.NewForConfig
	hfExplicitURL = hyperfleet.ExplicitURL
	hfExitFn      = func(code int) { os.Exit(code) }
)

type Runtime struct {
	Reporter         reporter.Logger
	Logger           *logrus.Logger
	OCMClient        *ocm.Client
	AWSClient        aws.Client
	AWSConfig        awssdk.Config
	Creator          *aws.Creator
	Region           string
	ClusterKey       string
	Cluster          *cmv1.Cluster
	Spinner          *spinner.Spinner
	HyperFleetClient hyperfleetclientset.Interface
}

func NewRuntime() *Runtime {
	r := reporter.CreateReporter()
	logger := logging.NewLogger()
	spinner := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	return &Runtime{Reporter: output.NewStructuredReporter(r), Logger: logger, Spinner: spinner}
}

// WithOCM Adds an OCM client to the runtime. Requires a deferred call to `.Cleanup()` to close connections.
func (r *Runtime) WithOCM() *Runtime {
	if r.OCMClient == nil {
		r.OCMClient = ocm.CreateNewClientOrExit(r.Logger, r.Reporter)
	}
	return r
}

// WithAWS Adds an AWS client to the runtime
func (r *Runtime) WithAWS() *Runtime {
	// dependency to ocm client to validate the region
	r.WithOCM()
	err := r.OCMClient.ValidateAwsClientRegion()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
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

// WithAWSOnly initializes the AWS client and creator without requiring an OCM connection.
// Use this when OCM credentials may not be available (e.g. hyperfleet-only mode).
func (r *Runtime) WithAWSOnly() *Runtime {
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

// WithAWSWarnInsteadOfExit Adds an AWS client to the runtime with no region validation
func (r *Runtime) WithAWSWarnInsteadOfExit() *Runtime {
	// dependency to ocm client to validate the region
	r.WithOCM()
	err := r.OCMClient.ValidateAwsClientRegion()
	if err != nil {
		r.Reporter.Warnf("%v", err)
	}
	if r.AWSClient == nil {
		r.AWSClient = aws.CreateNewClientOrExit(r.Logger, r.Reporter)
	}
	if r.Creator == nil {
		var err error
		r.Creator, err = r.AWSClient.GetCreator()
		if err != nil {
			_ = r.Reporter.Errorf("Failed to get AWS creator: %v", err)
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

// GetClusterKey Load the cluster key provided by the user into the runtime and return it
func (r *Runtime) GetClusterKey() string {
	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	r.ClusterKey = clusterKey
	return clusterKey
}

// WithHyperFleet builds the Platform API v2 client from the --hyperfleet-url flag.
// Region is resolved from --region / AWS_DEFAULT_REGION, falling back to extraction
// from the URL hostname. --profile / AWS_PROFILE are honoured for credential loading.
// No OCM login is required.
func (r *Runtime) WithHyperFleet() *Runtime {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rawURL := hfExplicitURL()

	if err := hyperfleet.ValidateURL(rawURL); err != nil {
		r.Reporter.Errorf("%v", err)
		hfExitFn(1)
		return r
	}

	// Resolve region: explicit flag/env takes precedence, then extracted from URL.
	region := arguments.GetRegion()
	if region == "" {
		var err error
		region, err = hyperfleet.ExtractRegion(rawURL)
		if err != nil {
			r.Reporter.Errorf("%v", err)
			hfExitFn(1)
			return r
		}
	} else {
		if err := hyperfleet.CheckRegionConflict(region, rawURL, r.Reporter); err != nil {
			r.Reporter.Errorf("%v", err)
			hfExitFn(1)
			return r
		}
	}

	// Load AWS config, honouring --profile / AWS_PROFILE and resolved region.
	awsCfg, err := awsLoadConfig(ctx, region, profile.Profile())
	if err != nil {
		r.Reporter.Errorf("Failed to load AWS config: %v", err)
		hfExitFn(1)
		return r
	}

	// Derive account ID and caller ARN via STS (no OCM required).
	identity, err := awsGetIdentity(ctx, awsCfg)
	if err != nil {
		r.Reporter.Errorf("Failed to get AWS caller identity: %v", err)
		hfExitFn(1)
		return r
	}
	creator, err := aws.CreatorForCallerIdentity(identity)
	if err != nil {
		r.Reporter.Errorf("Failed to build creator from caller identity: %v", err)
		hfExitFn(1)
		return r
	}
	r.Creator = creator
	r.AWSConfig = awsCfg
	r.Region = region

	cs, err := hfNewClient(&hfrest.Config{
		Host:      rawURL,
		Region:    region,
		AccountID: creator.AccountID,
		CallerARN: creator.ARN,
		AWSConfig: awsCfg,
	})
	if err != nil {
		r.Reporter.Errorf("Failed to build Platform API client: %v", err)
		hfExitFn(1)
		return r
	}
	r.HyperFleetClient = cs
	return r
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
