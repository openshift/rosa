package cluster

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/openshift-online/rosa-hyperfleet-api/clientset/wrappers"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

type clusterListItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	State    string `json:"state"`
	Topology string `json:"topology"`
}

var (
	hfEnabled      = hyperfleet.Enabled
	hfListClusters = func() {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleetList(r)
	}
)

func runHyperfleetList(r *rosa.Runtime) {
	ctx := context.Background()

	list, err := r.HyperFleetClient.HyperfleetV1alpha1().Clusters(r.Creator.AccountID).List(ctx, wrappers.ListOptions{})
	if err != nil {
		r.Reporter.Errorf("Failed to list clusters: %v", err)
		os.Exit(1)
	}

	if output.HasFlag() {
		items := make([]clusterListItem, 0, len(list.Items))
		for _, c := range list.Items {
			items = append(items, clusterListItem{
				ID:       string(c.UID),
				Name:     c.Name,
				State:    string(c.Status.Phase),
				Topology: "Hosted CP",
			})
		}
		if err := output.Print(items); err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		return
	}

	if len(list.Items) == 0 {
		r.Reporter.Infof("No clusters available")
		return
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "ID\tNAME\tSTATE\tTOPOLOGY\n")
	for _, c := range list.Items {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n",
			string(c.UID),
			c.Name,
			string(c.Status.Phase),
			"Hosted CP",
		)
	}
	writer.Flush()
}
