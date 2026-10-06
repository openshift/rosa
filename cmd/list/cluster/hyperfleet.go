package cluster

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"

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
	hfExitFn       = func(code int) { os.Exit(code) }
	hfListClusters = func() {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		if err := runHyperfleetList(r); err != nil {
			r.Reporter.Errorf("%v", err)
			hfExitFn(1)
		}
	}
)

func runHyperfleetList(r *rosa.Runtime) error {
	ctx := context.Background()

	list, err := r.HyperFleetClient.HyperfleetV1alpha1().Clusters().List(ctx, platform.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
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
			return fmt.Errorf("failed to print clusters: %w", err)
		}
		return nil
	}

	if len(list.Items) == 0 {
		r.Reporter.Infof("No clusters available")
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintf(writer, "ID\tNAME\tSTATE\tTOPOLOGY\n"); err != nil {
		return fmt.Errorf("failed to format cluster list header: %w", err)
	}
	for _, c := range list.Items {
		if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n",
			string(c.UID),
			c.Name,
			string(c.Status.Phase),
			"Hosted CP",
		); err != nil {
			return fmt.Errorf("failed to format cluster list: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to write cluster list: %w", err)
	}
	return nil
}
