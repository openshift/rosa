package accessrequests

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	v1 "github.com/openshift-online/ocm-sdk-go/accesstransparency/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use   = "access-request"
	short = "List Access Requests"
	long  = "List Access Requests in Pending or Approved status. " +
		"If '--cluster' flag is used, list all Access Requests in any status for the specified cluster."
	example = `  # List all Access Requests for cluster 'foo'
  rosa list access-request --cluster foo
  `
)

func NewListAccessRequestsCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:     use,
		Aliases: []string{"accessrequest", "accessrequests", "access-requests"},
		Short:   short,
		Long:    long,
		Example: example,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), ListAccessRequestsRunner()),
		Args:    cobra.NoArgs,
	}

	output.AddFlag(cmd)
	output.AddHideEmptyColumnsFlag(cmd)
	ocm.AddOptionalClusterFlag(cmd)
	return cmd
}

func ListAccessRequestsRunner() rosa.CommandRunner {
	return func(_ context.Context, r *rosa.Runtime, cmd *cobra.Command, _ []string) error {
		clusterId := ""
		if cmd.Flags().Changed("cluster") {
			cluster, err := r.OCMClient.GetCluster(r.GetClusterKey(), r.Creator)
			if err != nil {
				return err
			}
			clusterId = cluster.ID()
		}
		accessRequests, err := r.OCMClient.ListAccessRequest(clusterId)
		if err != nil {
			return err
		}
		if output.HasFlag() {
			output.Print(accessRequests)
		} else {
			if len(accessRequests) == 0 {
				if clusterId == "" {
					r.Reporter.Infof("There are no Access Requests in Pending or Approved status.")
				} else {
					r.Reporter.Infof("There are no Access Requests for cluster '%s'.", r.ClusterKey)
				}
				return nil
			}

			hasPending := false
			pendingId := "<ID>"
			for _, accessRequest := range accessRequests {
				if accessRequest.Status().State() == v1.AccessRequestStatePending ||
					accessRequest.Status().State() == v1.AccessRequestStateApproved {
					hasPending = true
					if clusterId != "" {
						pendingId = accessRequest.ID()
					}
				}
			}

			if hasPending {
				r.Reporter.Infof("Run the following command to approve or deny the Access Request:\n\n"+
					"   rosa create decision --access-request %s --decision Approved\n"+
					"   rosa create decision --access-request %s --decision Denied --justification \"justification\"\n",
					pendingId, pendingId)
			}

			headers := []string{"STATE", "ID", "CLUSTER ID", "UPDATED AT"}
			var tableData [][]string
			for _, accessRequest := range accessRequests {
				row := []string{
					string(accessRequest.Status().State()),
					accessRequest.ID(),
					accessRequest.ClusterId(),
					accessRequest.UpdatedAt().UTC().Format(time.UnixDate),
				}
				tableData = append(tableData, row)
			}

			writer := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

			if output.ShouldHideEmptyColumns() {
				newHeaders, newData := output.RemoveEmptyColumns(headers, tableData)
				config := output.TableConfig{
					Separator:            "\t",
					HasTrailingSeparator: false,
					UseFprintln:          false,
				}
				output.PrintTable(writer, newHeaders, newData, config)
			} else {
				fmt.Fprint(writer, "STATE\tID\tCLUSTER ID\tUPDATED AT\n")
				for _, row := range tableData {
					fmt.Fprintf(writer, "%s\n", strings.Join(row, "\t"))
				}
			}

			writer.Flush()
		}
		return nil
	}
}
