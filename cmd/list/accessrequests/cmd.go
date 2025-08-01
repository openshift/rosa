package accessrequests

import (
	"context"
	"fmt"
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
			// Create the writer that will be used to print the tabulated results:
			outputStr, hasPending, pendingId := getAccessRequestsOutput(clusterId, accessRequests)
			fmt.Print(outputStr)

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

func getAccessRequestsOutput(clusterId string, accessRequests []*v1.AccessRequest) (string, bool, string) {
	tb := output.NewTableBuilder()
	tb.SetHeaders("STATE", "ID", "CLUSTER ID", "UPDATED AT")

	hasPending := false
	id := "<ID>"
	for _, accessRequest := range accessRequests {
		if accessRequest.Status().State() == v1.AccessRequestStatePending {
			hasPending = true
			if clusterId != "" {
				id = accessRequest.ID()
			}
		}
		tb.AddRow(
			string(accessRequest.Status().State()),
			accessRequest.ID(),
			accessRequest.ClusterId(),
			accessRequest.UpdatedAt().Format(time.UnixDate),
		)
	}

	return tb.RenderToString(), hasPending, id
}
