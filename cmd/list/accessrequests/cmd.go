package accessrequests

import (
	"context"
	"os"
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

			hasPending, pendingId := checkForPendingRequests(clusterId, accessRequests)

			if hasPending {
				r.Reporter.Infof("Run the following command to approve or deny the Access Request:\n\n"+
					"   rosa create decision --access-request %s --decision Approved\n"+
					"   rosa create decision --access-request %s --decision Denied --justification \"justification\"\n",
					pendingId, pendingId)
			}
		}
		return nil
	}
}

func checkForPendingRequests(clusterId string, accessRequests []*v1.AccessRequest) (bool, string) {
	hasPending := false
	id := "<ID>"
	for _, accessRequest := range accessRequests {
		if accessRequest.Status().State() == v1.AccessRequestStatePending ||
			accessRequest.Status().State() == v1.AccessRequestStateApproved {
			hasPending = true
			if clusterId != "" {
				id = accessRequest.ID()
			}
			// Once we find the first pending/approved request for the cluster, we can break
			// since we only need one ID for the suggestion message
			if clusterId != "" && hasPending {
				break
			}
		}
	}
	return hasPending, id
}
