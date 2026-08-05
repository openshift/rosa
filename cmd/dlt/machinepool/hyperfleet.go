package machinepool

import (
	"os"

	"github.com/openshift-online/rosa-hyperfleet-api/clientset/wrappers"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hfEnabled           = hyperfleet.Enabled
	exitFn              = func(code int) { os.Exit(code) }
	confirmFn           = confirm.Confirm
	hfDeleteMachinePool = func(cmd *cobra.Command, userOptions *DeleteMachinepoolUserOptions, argv []string) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleetDelete(r, cmd, userOptions, argv)
	}
)

func runHyperfleetDelete(
	r *rosa.Runtime, cmd *cobra.Command, userOptions *DeleteMachinepoolUserOptions, argv []string,
) {
	ctx := cmd.Context()

	nodePoolName := userOptions.machinepool
	if nodePoolName == "" && len(argv) > 0 {
		nodePoolName = argv[0]
	}
	if nodePoolName == "" {
		r.Reporter.Errorf("--machinepool is required")
		exitFn(1)
	}

	clusterKey, err := ocm.GetClusterKey()
	if err != nil || clusterKey == "" {
		r.Reporter.Errorf("--cluster is required")
		exitFn(1)
	}

	clusterUID, err := hyperfleet.ResolveClusterUID(ctx, r.HyperFleetClient, r.Creator.AccountID, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
	}

	nodePoolID, err := hyperfleet.ResolveNodePoolUID(ctx, r.HyperFleetClient, clusterUID, nodePoolName)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
	}

	if !confirmFn("delete machine pool '%s' on cluster '%s'", nodePoolName, clusterKey) {
		return
	}

	if err = r.HyperFleetClient.HyperfleetV1alpha1().NodePools(clusterUID).
		Delete(ctx, nodePoolID, wrappers.DeleteOptions{}); err != nil {
		r.Reporter.Errorf("Failed to delete node pool '%s': %v", nodePoolName, err)
		exitFn(1)
	}

	r.Reporter.Infof("Node pool '%s' will start deleting now", nodePoolName)
}
