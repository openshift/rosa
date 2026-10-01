package machinepool

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
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
		r.Reporter.Errorf("you need to specify a machine pool name")
		exitFn(1)
	}

	clusterKey, err := ocm.GetClusterKey()
	if err != nil || clusterKey == "" {
		r.Reporter.Errorf("--cluster is required")
		exitFn(1)
	}

	clusterUID, err := hyperfleet.ResolveClusterUID(ctx, r.HyperFleetClient, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
	}

	nodePools, err := r.HyperFleetClient.HyperfleetV1alpha1().NodePools(clusterUID).
		List(ctx, platform.ListOptions{})
	if err != nil {
		r.Reporter.Errorf("Failed to list node pools for cluster '%s': %v", clusterKey, err)
		exitFn(1)
	}

	var np *v1alpha1.NodePool
	for i := range nodePools.Items {
		if nodePools.Items[i].Name == nodePoolName {
			np = &nodePools.Items[i]
			break
		}
	}
	if np == nil {
		r.Reporter.Errorf("Node pool '%s' not found in cluster '%s'", nodePoolName, clusterKey)
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
	diskSize := ""
	if np.Spec.NodePool.Platform.AWS != nil {
		instanceType = np.Spec.NodePool.Platform.AWS.InstanceType
		if np.Spec.NodePool.Platform.AWS.Subnet.ID != nil {
			subnetID = *np.Spec.NodePool.Platform.AWS.Subnet.ID
		}
		diskSize = hyperfleet.FormatNodePoolDiskSize(np.Spec.NodePool.Platform.AWS)
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
		"disk_size":    diskSize,
		"version":      np.Spec.NodePool.Release.Image,
		"created_at":   np.CreationTimestamp.UTC().Format(time.RFC3339),
		"conditions":   conditions,
	}
}

func hfNodePoolToString(np *v1alpha1.NodePool, clusterName string) string {
	instanceType := ""
	subnetID := ""
	diskSize := ""
	spotInstances := "No"
	securityGroupIDs := ""
	tags := ""

	if np.Spec.NodePool.Platform.AWS != nil {
		aws := np.Spec.NodePool.Platform.AWS
		instanceType = aws.InstanceType
		if aws.Subnet.ID != nil {
			subnetID = *aws.Subnet.ID
		}
		diskSize = hyperfleet.FormatNodePoolDiskSize(aws)

		// Spot instances
		if aws.Placement != nil && aws.Placement.MarketType == "Spot" {
			// MaxPrice can be empty, "on-demand", or an actual price
			if aws.Placement.Spot.MaxPrice != "" && aws.Placement.Spot.MaxPrice != "on-demand" {
				spotInstances = fmt.Sprintf("Yes (max $%s)", aws.Placement.Spot.MaxPrice)
			} else {
				spotInstances = "Yes (on-demand)"
			}
		}

		// Security groups
		if len(aws.SecurityGroups) > 0 {
			sgIDs := make([]string, 0, len(aws.SecurityGroups))
			for _, sg := range aws.SecurityGroups {
				if sg.ID != nil {
					sgIDs = append(sgIDs, *sg.ID)
				}
			}
			securityGroupIDs = strings.Join(sgIDs, ", ")
		}

		// Tags
		if len(aws.ResourceTags) > 0 {
			tagPairs := make([]string, 0, len(aws.ResourceTags))
			for _, tag := range aws.ResourceTags {
				tagPairs = append(tagPairs, fmt.Sprintf("%s=%s", tag.Key, tag.Value))
			}
			tags = strings.Join(tagPairs, ", ")
		}
	}

	// Autoscaling or fixed replicas (desired replicas)
	autoscaling := "No"
	desiredReplicasStr := ""
	if np.Spec.NodePool.AutoScaling != nil {
		autoscaling = "Yes"
		minVal := int32(0)
		if np.Spec.NodePool.AutoScaling.Min != nil {
			minVal = *np.Spec.NodePool.AutoScaling.Min
		}
		maxVal := np.Spec.NodePool.AutoScaling.Max
		desiredReplicasStr = fmt.Sprintf("\n - Min replicas: %d\n - Max replicas: %d", minVal, maxVal)
	} else if np.Spec.NodePool.Replicas != nil {
		desiredReplicasStr = fmt.Sprintf("%d", *np.Spec.NodePool.Replicas)
	}

	// Current replicas from status. A nil value means the API has not reported
	// an observed replica count yet; zero is a valid reported value.
	currentReplicasStr := ""
	if np.Status.Replicas != nil {
		currentReplicasStr = fmt.Sprintf("%d", *np.Status.Replicas)
	}

	// Labels
	labels := ""
	if len(np.Spec.NodePool.NodeLabels) > 0 {
		labelPairs := make([]string, 0, len(np.Spec.NodePool.NodeLabels))
		for k, v := range np.Spec.NodePool.NodeLabels {
			labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, v))
		}
		labels = strings.Join(labelPairs, ", ")
	}

	// Taints
	taintsStr := ""
	if len(np.Spec.NodePool.Taints) > 0 {
		taintStrs := make([]string, 0, len(np.Spec.NodePool.Taints))
		for _, t := range np.Spec.NodePool.Taints {
			taintStrs = append(taintStrs, fmt.Sprintf("%s=%s:%s", t.Key, t.Value, t.Effect))
		}
		taintsStr = strings.Join(taintStrs, ", ")
	}

	// Autorepair
	autorepair := "No"
	if np.Spec.NodePool.Management.AutoRepair {
		autorepair = "Yes"
	}

	// Tuning configs
	tuningConfigs := ""
	if len(np.Spec.NodePool.TuningConfig) > 0 {
		configNames := make([]string, 0, len(np.Spec.NodePool.TuningConfig))
		for _, cfg := range np.Spec.NodePool.TuningConfig {
			configNames = append(configNames, cfg.Name)
		}
		tuningConfigs = strings.Join(configNames, ", ")
	}

	// Kubelet configs
	kubeletConfigs := ""
	if len(np.Spec.NodePool.Config) > 0 {
		configNames := make([]string, 0, len(np.Spec.NodePool.Config))
		for _, cfg := range np.Spec.NodePool.Config {
			configNames = append(configNames, cfg.Name)
		}
		kubeletConfigs = strings.Join(configNames, ", ")
	}

	// Node drain timeout
	nodeDrainTimeout := ""
	if np.Spec.NodePool.NodeDrainTimeout != nil {
		nodeDrainTimeout = np.Spec.NodePool.NodeDrainTimeout.Duration.String()
	}

	s := fmt.Sprintf("\n"+
		"Name:                                  %s\n"+
		"ID:                                    %s\n"+
		"Cluster:                               %s\n"+
		"State:                                 %s\n"+
		"Autoscaling:                           %s\n"+
		"Desired replicas:                      %s\n"+
		"Current replicas:                      %s\n"+
		"Instance type:                         %s\n"+
		"Labels:                                %s\n"+
		"Tags:                                  %s\n"+
		"Taints:                                %s\n"+
		"Availability zone:                     %s\n"+
		"Subnet:                                %s\n"+
		"Spot instances:                        %s\n"+
		"Disk Size:                             %s\n"+
		"Version:                               %s\n"+
		"EC2 Metadata Http Tokens:              %s\n"+
		"Autorepair:                            %s\n"+
		"Tuning configs:                        %s\n"+
		"Kubelet configs:                       %s\n"+
		"Additional security group IDs:         %s\n"+
		"Node drain grace period:               %s\n"+
		"Created:                               %s\n",
		np.Name,
		string(np.UID),
		clusterName,
		string(np.Status.Phase),
		autoscaling,
		desiredReplicasStr,
		currentReplicasStr,
		instanceType,
		labels,
		tags,
		taintsStr,
		"", // Availability zone - needs investigation where this comes from
		subnetID,
		spotInstances,
		diskSize,
		np.Spec.NodePool.Release.Image,
		"", // EC2 Metadata Http Tokens - TODO: add support
		autorepair,
		tuningConfigs,
		kubeletConfigs,
		securityGroupIDs,
		nodeDrainTimeout,
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
