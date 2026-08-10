package version

import (
	"fmt"
	"runtime/debug"

	verify "github.com/openshift/rosa/cmd/verify/rosa"
	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/version"
)

type RosaVersionUserOptions struct {
	clientOnly bool
	verbose    bool
	build      bool
}

func NewRosaVersionUserOptions() *RosaVersionUserOptions {
	return &RosaVersionUserOptions{}
}

type RosaVersionOptions struct {
	reporter      *reporter.Object
	verifyRosa    verify.VerifyRosa
	readBuildInfo func() (*debug.BuildInfo, bool)

	args *RosaVersionUserOptions
}

func NewRosaVersionOptions() (*RosaVersionOptions, error) {
	verifyRosa, err := verify.NewVerifyRosaOptions()
	if err != nil {
		return nil, fmt.Errorf("failed to build rosa verify options : %v", err)
	}

	return &RosaVersionOptions{
		verifyRosa:    verifyRosa,
		reporter:      reporter.CreateReporter(),
		readBuildInfo: debug.ReadBuildInfo,
		args:          NewRosaVersionUserOptions(),
	}, nil
}

func (o *RosaVersionOptions) Version() error {
	o.reporter.Infof("%s", info.DefaultVersion)

	if o.args.verbose {
		o.reporter.Infof("Information and download locations:\n\t%s\n\t%s\n",
			version.ConsoleLatestFolder,
			version.DownloadLatestMirrorFolder)
	}

	if o.args.build {
		buildInfo, ok := o.readBuildInfo()
		if ok {
			var revision string

			for _, setting := range buildInfo.Settings {
				if setting.Key == "vcs.revision" {
					revision = setting.Value
				}
			}

			if revision == "" {
				revision = info.Build
			}
			o.reporter.Infof("Git commit: %s", revision)
		} else {
			o.reporter.Infof("Git commit: %s", info.Build)
		}
	}

	if !o.args.clientOnly {
		if err := o.verifyRosa.Verify(); err != nil {
			return fmt.Errorf("failed to verify rosa : %v", err)
		}
	}
	return nil
}

func (o *RosaVersionOptions) BindAndValidate(options *RosaVersionUserOptions) {
	o.args = options
}
