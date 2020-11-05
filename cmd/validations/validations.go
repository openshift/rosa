package validations

import (
	"os"

	"github.com/openshift/moactl/pkg/aws"
	"github.com/openshift/moactl/pkg/logging"
	rprtr "github.com/openshift/moactl/pkg/reporter"
	"github.com/spf13/cobra"
)

// Validations will validate if CF stack/users exist
func Validations(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)
	// Create the AWS client:
	client, err := aws.NewClient().
		Logger(logger).
		Region(aws.DefaultRegion).
		Build()
	if err != nil {
		reporter.Errorf("Error creating AWS client: %v", err)
		os.Exit(1)
	}

	reporter.Debugf("Validating cloudformation stack exists")
	stackExist, _, err := client.CheckStackReadyOrNotExisting(aws.OsdCcsAdminStackName)
	if !stackExist || err != nil {
		reporter.Errorf("Cloudformation stack does not exist. Run `rosa init` first")
		os.Exit(1)
	}
	reporter.Debugf("cloudformation stack is valid!")
}
