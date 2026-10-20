package cluster

import (
	"fmt"
	"os"
	"time"

	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/seaplane"
)

const (
	seaplaneWatchInterval = 30 * time.Second
	seaplaneWatchTimeout  = 90 * time.Minute
)

var (
	exitFn                = func(code int) { os.Exit(code) }
	confirmFn             = confirm.Confirm
	seaplaneDeleteCluster = func(cmd *cobra.Command) {
		r := rosa.NewRuntime().WithSeaplane()
		defer r.Cleanup()
		runSeaplaneDelete(r, cmd)
	}
)

// runSeaplaneDelete deletes an HCP cluster via the Platform API v2.
// It resolves a user-provided name or UID to metadata.name for account-scoped routes.
func runSeaplaneDelete(r *rosa.Runtime, cmd *cobra.Command) {
	ctx := cmd.Context()

	clusterKey, err := ocm.GetClusterKey()
	if err != nil || clusterKey == "" {
		r.Reporter.Errorf("--cluster is required")
		exitFn(1)
	}

	clusterName, err := seaplane.ResolveClusterName(ctx, r.SeaplaneClient, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
	}

	clusters := r.SeaplaneClient.SeaplaneV1alpha1().Clusters()
	cluster, err := clusters.Get(ctx, clusterName, platform.GetOptions{})
	if err != nil {
		r.Reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		exitFn(1)
	}

	if err := ensureSeaplaneDeleteProtectionDisabled(cluster, clusterKey); err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
	}

	if !confirmFn("delete cluster %s", clusterKey) {
		return
	}

	if err = clusters.Delete(ctx, clusterName, platform.DeleteOptions{}); err != nil {
		r.Reporter.Errorf("Failed to delete cluster '%s': %v", clusterKey, err)
		exitFn(1)
	}

	r.Reporter.Infof("Cluster '%s' will start deleting now", clusterKey)

	if !args.watch {
		return
	}

	err = clusters.WaitUntil(ctx, clusterName, func(c *v1alpha1.Cluster) bool {
		return c == nil
	}, seaplaneWatchInterval, seaplaneWatchTimeout)
	if err != nil {
		r.Reporter.Errorf("Failed to watch cluster deletion: %v", err)
		exitFn(1)
	}
	r.Reporter.Infof("Cluster '%s' deleted", clusterKey)
}

// ensureSeaplaneDeleteProtectionDisabled returns an error if delete protection is enabled.
func ensureSeaplaneDeleteProtectionDisabled(cluster *v1alpha1.Cluster, clusterKey string) error {
	if cluster.Spec.DeleteProtection != nil && *cluster.Spec.DeleteProtection {
		return fmt.Errorf(
			"delete protection is active on cluster '%s', "+
				"to disable it run 'rosa edit cluster -c %s --delete-protection=false'",
			clusterKey, clusterKey)
	}
	return nil
}
