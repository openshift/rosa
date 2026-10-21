package machinepool

import (
	"os"

	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive/confirm"
	rosamachinepool "github.com/openshift/rosa/pkg/machinepool"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/seaplane"
)

var (
	exitFn                    = func(code int) { os.Exit(code) }
	confirmFn                 = confirm.Confirm
	seaplaneDeleteMachinePool = func(cmd *cobra.Command, userOptions *DeleteMachinepoolUserOptions, argv []string) {
		r := rosa.NewRuntime().WithSeaplane()
		defer r.Cleanup()
		runSeaplaneDelete(r, cmd, userOptions, argv)
	}
)

func runSeaplaneDelete(
	r *rosa.Runtime, cmd *cobra.Command, userOptions *DeleteMachinepoolUserOptions, argv []string,
) {
	ctx := cmd.Context()

	nodePoolName := userOptions.machinepool
	if nodePoolName == "" && len(argv) > 0 {
		nodePoolName = argv[0]
	}
	if nodePoolName == "" {
		r.Reporter.Errorf("you need to specify a machine pool name")
		exitFn(1)
	}
	if !rosamachinepool.MachinePoolKeyRE.MatchString(nodePoolName) {
		r.Reporter.Errorf("expected a valid identifier for the machine pool")
		exitFn(1)
	}

	clusterKey, err := ocm.GetClusterKey()
	if err != nil || clusterKey == "" {
		r.Reporter.Errorf("--cluster is required")
		exitFn(1)
	}

	clusterUID, err := seaplane.ResolveClusterUID(ctx, r.SeaplaneClient, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
	}

	resourceName, err := seaplane.ResolveNodePoolName(ctx, r.SeaplaneClient, clusterUID, nodePoolName)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
	}

	if !confirmFn("delete machine pool '%s' on cluster '%s'", nodePoolName, clusterKey) {
		return
	}

	if err = r.SeaplaneClient.SeaplaneV1alpha1().NodePools().
		Delete(ctx, resourceName, platform.DeleteOptions{}); err != nil {
		r.Reporter.Errorf("Failed to delete node pool '%s': %v", nodePoolName, err)
		exitFn(1)
	}

	r.Reporter.Infof("Node pool '%s' will start deleting now", nodePoolName)
}
