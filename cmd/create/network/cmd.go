package network

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
	helper "github.com/openshift/rosa/pkg/network"
	"github.com/openshift/rosa/pkg/ocm"
	opts "github.com/openshift/rosa/pkg/options/network"
	"github.com/openshift/rosa/pkg/rosa"
)

const defaultTemplate = "rosa-quickstart-default-vpc"

func NewNetworkCommand() *cobra.Command {
	cmd, options := opts.BuildNetworkCommandWithOptions()
	cmd.Run = rosa.DefaultRunner(rosa.RuntimeWithOCMAndAWS(), NetworkRunner(options))
	interactive.AddModeFlag(cmd)

	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		templateDir := options.TemplateDir
		err := filepath.WalkDir(templateDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(d.Name(), ".yaml") {
				templateBody, err := os.ReadFile(path)
				if err != nil {
					fmt.Println(err)
					return nil
				}

				var templateMap map[string]interface{}
				err = yaml.Unmarshal(templateBody, &templateMap)
				if err != nil {
					fmt.Println(err)
					return nil
				}

				parameters, ok := templateMap["Parameters"].(map[string]interface{})
				if !ok {
					fmt.Printf("No parameters found in the CloudFormation template %s\n", d.Name())
					return nil
				}

				fmt.Printf("Available parameters in %s/%s:\n", filepath.Base(filepath.Dir(path)), d.Name())
				for paramName := range parameters {
					fmt.Printf("  %s\n", paramName)
				}
				fmt.Printf("  %s\n", "Tags")
			}
			return nil
		})
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println("\n" + cmd.UsageString())
	})

	return cmd
}

func NetworkRunner(userOptions *opts.NetworkUserOptions) rosa.CommandRunner {
	return func(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command, argv []string) error {
		var err error
		var templateFile string
		var templateBody []byte
		templateCommand := defaultTemplate
		options := NewNetworkOptions()
		userOptions.CleanTemplateDir()
		options.Bind(userOptions)

		defer r.Cleanup()

		orgID, _, err := r.OCMClient.GetCurrentOrganization()
		if err != nil {
			return err
		}

		parsedParams, parsedTags, err := helper.ParseParams(userOptions.Params)
		if err != nil {
			return err
		}

		if parsedParams["Name"] == "" {
			r.Reporter.Infof("Name not provided, using default name %s", r.Creator.AccountID)
			parsedParams["Name"] = "rosa-network-stack-" + r.Creator.AccountID
		}
		if parsedParams["Region"] == "" {
			r.Reporter.Infof("Region not provided, using default region %s", r.AWSClient.GetRegion())
			parsedParams["Region"] = r.AWSClient.GetRegion()
		}

		err = extractTemplateCommand(r, argv, options.args,
			&templateCommand, &templateFile)
		if err != nil {
			return err
		}

		service := helper.NewNetworkService()

		mode, err := interactive.GetMode()
		if err != nil {
			return err
		}

		switch mode {
		case interactive.ModeManual:
			r.Reporter.Infof(helper.ManualModeHelperMessage(parsedParams, templateFile, parsedTags))
			r.OCMClient.LogEvent("RosaNetworkStackManual",
				map[string]string{
					ocm.Account:      r.Creator.AccountID,
					ocm.Organization: orgID,
					"template":       templateFile,
				},
			)
			return nil
		default:
			r.OCMClient.LogEvent("RosaNetworkStack",
				map[string]string{
					ocm.Account:      r.Creator.AccountID,
					ocm.Organization: orgID,
					"template":       templateFile,
				},
			)
			return service.CreateStack(&templateFile, &templateBody, parsedParams, parsedTags)
		}
	}
}

func extractTemplateCommand(r *rosa.Runtime, argv []string, options *opts.NetworkUserOptions,
	templateCommand *string, templateFile *string) error {
	if len(argv) == 0 {
		r.Reporter.Infof("No template name provided in the command. "+
			"Defaulting to %s. Please note that a corresponding directory with this name"+
			" must exist under the specified path <`--template-dir`> or the templates directory"+
			" for the command to work correctly. ", *templateCommand)
		*templateCommand = defaultTemplate
		*templateFile = CloudFormationTemplateFile
	}

	for _, arg := range argv {
		if !strings.HasPrefix(arg, "--param") {
			*templateCommand = arg
			break
		}
	}
	if *templateCommand == defaultTemplate {
		*templateFile = CloudFormationTemplateFile
	} else {
		templateDir := options.TemplateDir
		*templateFile = helper.SelectTemplate(templateDir, *templateCommand)
		templateBody, err := os.ReadFile(*templateFile)
		if err != nil {
			return fmt.Errorf("failed to read template file: %w", err)
		}
		*templateFile = string(templateBody)
	}
	return nil
}
