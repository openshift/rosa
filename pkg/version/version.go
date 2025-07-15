package version

import (
	"fmt"
	"net/http"
	"slices"

	goVer "github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/cache"
	"github.com/openshift/rosa/pkg/clients"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/output"
)

const (
	DownloadLatestMirrorFolder = "https://mirror.openshift.com/pub/openshift-v4/clients/rosa/latest/"
	baseReleasesFolder         = "https://mirror.openshift.com/pub/openshift-v4/clients/rosa/"
	ConsoleLatestFolder        = "https://console.redhat.com/openshift/downloads#tool-rosa"
)

//go:generate mockgen -source=version.go -package=version -destination=./version_mock.go
type RosaVersion interface {
	IsLatest(latestVersion string) (*goVer.Version, bool, error)
}

var _ RosaVersion = &rosaVersion{}

func NewRosaVersion() (RosaVersion, error) {
	logger := logging.NewLogger()
	transport := http.DefaultTransport
	if logger.IsLevelEnabled(logrus.DebugLevel) {
		dumper, err := logging.NewRoundTripper().Logger(logger).Next(transport).Build()
		if err != nil {
			return &rosaVersion{}, fmt.Errorf("failed to create logger: %v", err)
		}
		transport = dumper
	}

	c, err := cache.NewRosaCacheService()
	if err != nil {
		return &rosaVersion{}, fmt.Errorf("failed to create cache service: %v", err)
	}

	return &rosaVersion{
		logger: logger,
		client: clients.NewDefaultHTTPClient(&http.Client{
			Transport: transport,
		}),
		retriever: NewRetriever(RetrieverSpec{
			Logger: logger,
			Client: clients.NewDefaultHTTPClient(&http.Client{
				Transport: transport,
			}),
			Cache: c,
		}),
	}, nil
}

type rosaVersion struct {
	logger    *logrus.Logger
	client    clients.HTTPClient
	retriever Retriever
}

func (v rosaVersion) IsLatest(version string) (*goVer.Version, bool, error) {
	currentVersion, err := goVer.NewVersion(version)
	if err != nil {
		return nil, false, fmt.Errorf("failed to retrieve current version: %v", err)
	}

	latestVersionFromMirror, err := v.retriever.RetrieveLatestVersionFromMirror()
	if err != nil {
		return nil, false, fmt.Errorf("failed to retrieve latest version from mirror: %v", err)
	}

	if currentVersion.LessThan(latestVersionFromMirror) {
		return latestVersionFromMirror, false, nil
	}

	return nil, true, nil
}

func ShouldRunCheck(cmd *cobra.Command) bool {
	return !slices.Contains([]string{"version", "rosa-client"}, cmd.Use) && !output.HasFlag()
}
