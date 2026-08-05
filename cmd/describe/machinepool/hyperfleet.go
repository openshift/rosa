package machinepool

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/openshift-online/rosa-hyperfleet-api/clientset/wrappers"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/hyperfleet-operator/api/v1alpha1"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hfEnabled             = hyperfleet.Enabled
	exitFn                = func(code int) { os.Exit(code) }
	hfDescribeMachinePool = func(userOptions *DescribeMachinepoolUserOptions, argv []string) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleetDescribe(r, userOptions, argv)
	}
)

func runHyperfleetDescribe(r *rosa.Runtime, userOptions *DescribeMachinepoolUserOptions, argv []string) {
	ctx := context.Background()

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

	np, err := r.HyperFleetClient.HyperfleetV1alpha1().NodePools(clusterUID).Get(ctx, nodePoolID, wrappers.GetOptions{})
	if err != nil {
		r.Reporter.Errorf("Failed to get node pool '%s': %v", nodePoolName, err)
		exitFn(1)
	}

	if output.HasFlag() {
		m := hfNodePoolToMap(np)
		if err := output.Print(m); err != nil {
			r.Reporter.Errorf("%s", err)
			exitFn(1)
		}
		return
	}

	fmt.Print(hfNodePoolToString(np, clusterKey))
}

func hfNodePoolToMap(np *v1alpha1.NodePool) map[string]interface{} {
	replicas := int32(0)
	if np.Spec.NodePool.Replicas != nil {
		replicas = *np.Spec.NodePool.Replicas
	}
	instanceType := ""
	subnetID := ""
	if np.Spec.NodePool.Platform.AWS != nil {
		instanceType = np.Spec.NodePool.Platform.AWS.InstanceType
		if np.Spec.NodePool.Platform.AWS.Subnet.ID != nil {
			subnetID = *np.Spec.NodePool.Platform.AWS.Subnet.ID
		}
	}

	conditions := make([]map[string]interface{}, 0, len(np.Status.Conditions))
	for _, cond := range np.Status.Conditions {
		conditions = append(conditions, map[string]interface{}{
			"type":    cond.Type,
			"status":  string(cond.Status),
			"reason":  cond.Reason,
			"message": cond.Message,
		})
	}

	return map[string]interface{}{
		"id":           string(np.UID),
		"name":         np.Name,
		"state":        string(np.Status.Phase),
		"replicas":     replicas,
		"instanceType": instanceType,
		"subnet":       subnetID,
		"version":      np.Spec.NodePool.Release.Image,
		"created_at":   np.CreationTimestamp.UTC().Format(time.RFC3339),
		"conditions":   conditions,
	}
}

func hfNodePoolToString(np *v1alpha1.NodePool, clusterName string) string {
	replicas := int32(0)
	if np.Spec.NodePool.Replicas != nil {
		replicas = *np.Spec.NodePool.Replicas
	}
	instanceType := ""
	subnetID := ""
	if np.Spec.NodePool.Platform.AWS != nil {
		instanceType = np.Spec.NodePool.Platform.AWS.InstanceType
		if np.Spec.NodePool.Platform.AWS.Subnet.ID != nil {
			subnetID = *np.Spec.NodePool.Platform.AWS.Subnet.ID
		}
	}

	s := fmt.Sprintf("\n"+
		"Name:                       %s\n"+
		"ID:                         %s\n"+
		"Cluster:                    %s\n"+
		"State:                      %s\n"+
		"Replicas:                   %d\n"+
		"Instance Type:              %s\n"+
		"Subnet:                     %s\n"+
		"Version:                    %s\n"+
		"Created:                    %s\n",
		np.Name,
		string(np.UID),
		clusterName,
		string(np.Status.Phase),
		replicas,
		instanceType,
		subnetID,
		np.Spec.NodePool.Release.Image,
		np.CreationTimestamp.UTC().Format("2006-01-02 15:04:05 UTC"),
	)

	if len(np.Status.Conditions) > 0 {
		s += "Conditions:\n"
		for _, cond := range np.Status.Conditions {
			s += fmt.Sprintf(" - %-10s %-5s  %s\n",
				cond.Type+":", string(cond.Status),
				hfConditionSummary(cond.Reason, cond.Message))
		}
	}

	return s
}

func hfConditionSummary(reason, message string) string {
	if reason == "" {
		return message
	}
	if message == "" || strings.HasPrefix(message, reason) {
		return reason
	}
	return reason + ": " + message
}
