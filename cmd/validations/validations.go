package validations

import (
	"os"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/spf13/cobra"
)

// Validations will validate if CF stack/users exist
func Validations(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	client := aws.GetAWSClientForUserRegion(reporter, logger)

	reporter.Debugf("Validating cloudformation stack exists")
	stackExist, _, err := client.CheckStackReadyOrNotExisting(aws.OsdCcsAdminStackName)
	if !stackExist || err != nil {
		reporter.Errorf("Cloudformation stack does not exist. Run `rosa init` first")
		os.Exit(1)
	}
	reporter.Debugf("cloudformation stack is valid!")
}
