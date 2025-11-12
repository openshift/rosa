package breakglasscredential

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/externalauthprovider"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "break-glass-credentials",
	Aliases: []string{"break-glass-credential", "breakglasscredential", "breakglasscredentials"},
	Short:   "List break glass credential",
	Long:    "List break glass credential for a cluster.",
	Example: `  # List all break glass credentials for a cluster named 'mycluster'"
  rosa list break-glass-credentials -c mycluster`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	ocm.AddClusterFlag(Cmd)
	output.AddFlag(Cmd)
	output.AddHideEmptyColumnsFlag(Cmd)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()
	err := runWithRuntime(r, cmd)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}
}

func runWithRuntime(r *rosa.Runtime, cmd *cobra.Command) error {
	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	externalAuthService := externalauthprovider.NewExternalAuthService(r.OCMClient)
	err := externalAuthService.IsExternalAuthProviderSupported(cluster, clusterKey)
	if err != nil {
		return err
	}

	// Load any break glass credential for this cluster
	r.Reporter.Debugf("Loading break glass credentials for cluster '%s'", clusterKey)
	breakGlassCredentials, err := r.OCMClient.GetBreakGlassCredentials(cluster.ID())
	if err != nil {
		return fmt.Errorf("failed to get break glass credentials for cluster '%s': %v", clusterKey, err)
	}

	if output.HasFlag() {
		err = output.Print(breakGlassCredentials)
		if err != nil {
			return fmt.Errorf("%s", err)
		}
		return nil
	}

	if len(breakGlassCredentials) == 0 {
		r.Reporter.Infof("There are no break glass credentials for cluster '%s'", clusterKey)
		return nil
	}

	headers := []string{"ID", "USERNAME", "STATUS"}
	var tableData [][]string
	for _, credential := range breakGlassCredentials {
		row := []string{
			credential.ID(),
			credential.Username(),
			string(credential.Status()),
		}
		tableData = append(tableData, row)
	}

	if output.ShouldHideEmptyColumns() {
		tableData = output.RemoveEmptyColumns(headers, tableData)
	} else {
		tableData = append([][]string{headers}, tableData...)
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	output.BuildTable(writer, "\t", tableData)

	if err := writer.Flush(); err != nil {
		return err
	}
	return nil
}
