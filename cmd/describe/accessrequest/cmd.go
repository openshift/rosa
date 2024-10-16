package accessrequest

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/openshift-online/ocm-sdk-go/accesstransparency/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "access-request"
	short   = "Show details of an Access Request"
	long    = short
	example = `  # Describe an Access Request wit id <access_request_id>
  rosa describe access-request --id <access_request_id>
  `
)

type Options struct {
	id string
}

func NewOptions() *Options {
	return &Options{}
}

func NewDescribeAccessRequestCommand() *cobra.Command {
	options := NewOptions()
	cmd := &cobra.Command{
		Use:     use,
		Aliases: []string{"accessrequest"},
		Short:   short,
		Long:    long,
		Example: example,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), DescribeAccessRequestRunner(options)),
		Args:    cobra.NoArgs,
	}
	flags := cmd.Flags()
	flags.StringVar(
		&options.id,
		"id",
		"",
		"ID of the Access Request. (required).",
	)
	cmd.MarkFlagRequired("id")
	output.AddFlag(cmd)
	return cmd
}

func DescribeAccessRequestRunner(options *Options) rosa.CommandRunner {
	return func(_ context.Context, r *rosa.Runtime, _ *cobra.Command, _ []string) error {
		accessRequest, exists, err := r.OCMClient.GetAccessRequest(options.id)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("The Access Request with id '%s' does not exist", options.id)
		}

		if output.HasFlag() {
			return output.Print(accessRequest)
		}
		fmt.Print(printAccessRequest(accessRequest))
		if accessRequest.Status().State() == v1.AccessRequestStatePending {
			r.Reporter.Infof("Run the following command to approve or deny the Access Request:\n\n"+
				"   rosa create decision --access-request %s --decision Approved\n"+
				"   rosa create decision --access-request %s --decision Denied --justification \"justification\"\n",
				accessRequest.ID(), accessRequest.ID())
		}
		return nil
	}
}

func printAccessRequest(accessRequest *v1.AccessRequest) string {
	outputMsg := fmt.Sprintf("\n"+
		"ID:                                %s\n"+
		"Subscription ID:                   %s\n"+
		"Cluster ID:                        %s\n"+
		"Support Case ID:                   %s\n"+
		"Requested By:                      %s\n"+
		"Created At:                        %s\n"+
		"Respond By:                        %s\n"+
		"Request Duration:                  %s\n"+
		"Justification:                     %s\n"+
		"Status:                            %s\n",
		accessRequest.ID(),
		accessRequest.SubscriptionId(),
		accessRequest.ClusterId(),
		accessRequest.SupportCaseId(),
		accessRequest.RequestedBy(),
		accessRequest.CreatedAt().Format(time.UnixDate),
		accessRequest.DeadlineAt().Format(time.UnixDate),
		accessRequest.Duration(),
		accessRequest.Justification(),
		accessRequest.Status().State(),
	)
	if len(accessRequest.Decisions()) > 0 {
		outputMsg = outputMsg + "Decisions:                           \n"
		for i := len(accessRequest.Decisions()) - 1; i >= 0; i-- {
			outputMsg = outputMsg + printDecision(accessRequest.Decisions()[i])
		}
	}
	return outputMsg
}

func printDecision(decision *v1.Decision) string {
	return fmt.Sprintf(
		"  - Decision:                      %s\n"+
			"    Decided By:                    %s\n"+
			"    Created At:                    %s\n"+
			"    Justification:                 %s\n",
		decision.Decision(),
		decision.DecidedBy(),
		decision.CreatedAt().Format(time.UnixDate),
		decision.Justification(),
	)
}
