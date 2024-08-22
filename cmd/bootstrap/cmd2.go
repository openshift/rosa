package bootstrap

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/bootstrap"
	"github.com/openshift/rosa/pkg/ocm"
	bsOpts "github.com/openshift/rosa/pkg/options/bootstrap"
	"github.com/openshift/rosa/pkg/rosa"
)

type BootstrapSpec struct {
	Service bootstrap.BootstrapService
}

type bootstrapStruct struct {
	service bootstrap.BootstrapService
}

func NewBootstrap(spec BootstrapSpec) bootstrapStruct {
	return bootstrapStruct{
		service: spec.Service,
	}
}

func NewBootstrapCommand() *cobra.Command {
	cmd, options := bsOpts.BuildBootstrapCommandWithOptions()
	cmd.Run = rosa.DefaultRunner(rosa.RuntimeWithOCMAndAWS(), BootstrapRunner(options))
	return cmd
}

func BootstrapRunner(userOptions *bsOpts.BootstrapUserOptions) rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, argv []string) error {
		var err error
		options := NewBootstrapOptions()
		options.args = userOptions

		defer r.Cleanup()

		orgID, _, err := r.OCMClient.GetCurrentOrganization()
		if err != nil {
			return r.Reporter.Errorf(err.Error())
		}

		parsedParams, parsedTags := bootstrap.ParseParams(userOptions.Params)

		if parsedParams["Name"] == "" {
			r.Logger.Debugf("Name not provided, using default name %s", r.Creator.AccountID)
			parsedParams["Name"] = "rosa-bootstrap-stack-" + r.Creator.AccountID
		}
		if parsedParams["Region"] == "" {
			r.Logger.Debugf("Region not provided, using default region %s", r.AWSClient.GetRegion())
			parsedParams["Region"] = r.AWSClient.GetRegion()
		}

		// Extract the first non-`--param` argument to use as the template command
		var templateCommand string
		for _, arg := range argv {
			if !strings.HasPrefix(arg, "--param") {
				templateCommand = arg
				break
			}
		}

		templateFile := bootstrap.SelectTemplate(templateCommand)
		if templateFile == "" {
			return r.Reporter.Errorf("No suitable template found")
		}

		r.OCMClient.LogEvent("RosaBootstrapStack",
			map[string]string{
				ocm.Account:      r.Creator.AccountID,
				ocm.Organization: orgID,
				"template":       templateFile,
			},
		)

		newService := NewBootstrap(BootstrapSpec{
			Service: bootstrap.NewBootstrapService(),
		})

		return newService.service.CreateStack(templateFile, parsedParams, parsedTags)
	}
}
