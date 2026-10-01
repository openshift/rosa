package machinepool

import (
	"context"
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	rosaaws "github.com/openshift/rosa/pkg/aws"
	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/hyperfleet"
	hfpathbind "github.com/openshift/rosa/pkg/hyperfleet/pathbind"
	"github.com/openshift/rosa/pkg/ocm"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/rosa"
)

// clusterNamespacePrefix is the prefix the Platform API requires on the
// metadata.namespace field of NodePool resources ("cluster-<uuid>").
// TODO: this derivation ideally belongs in the SDK (clientset/pathbind or
// the bridge wrapper) so consumers don't need to know the namespace format.
const clusterNamespacePrefix = "cluster-"

const maxAWSResourceTags = 25

// hfNodePoolInput is the backing store for hyperfleet-specific create machinepool flags.
var hfNodePoolInput hfpathbind.NodePoolCreateInput

var (
	exitFn = func(code int) { os.Exit(code) }

	hfCreateMachinePool = func(userOptions *mpOpts.CreateMachinepoolUserOptions, argv []string, cmd *cobra.Command) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()

		handler := &hyperfleetNodePoolCreate{
			userOptions: userOptions,
			argv:        argv,
		}
		if err := handler.validateAndParseUserOptions(); err != nil {
			r.Reporter.Errorf("%v", err)
			exitFn(1)
			return
		}

		clusterKey, err := ocm.GetClusterKey()
		if err != nil || clusterKey == "" {
			r.Reporter.Errorf("--cluster is required")
			exitFn(1)
			return
		}
		clusterUID, err := hyperfleet.ResolveClusterUID(context.Background(), r.HyperFleetClient, clusterKey)
		if err != nil {
			r.Reporter.Errorf("%v", err)
			exitFn(1)
			return
		}

		handler.clusterKey = clusterKey
		handler.clusterUID = clusterUID
		if err := hfpathbind.RunCreateNodePool(
			context.Background(),
			r,
			cmd,
			&hfNodePoolInput,
			handler,
			clusterNamespacePrefix+clusterUID,
		); err != nil {
			r.Reporter.Errorf("%v", err)
			exitFn(1)
		}
	}
)

// runHyperfleetCreate is a thin wrapper for direct test invocation without a real cobra.Command.
func runHyperfleetCreate(r *rosa.Runtime, userOptions *mpOpts.CreateMachinepoolUserOptions, argv []string) {
	// Validate name first so tests that don't mock List don't get unexpected calls.
	name := userOptions.Name
	if name == "" && len(argv) > 0 {
		name = argv[0]
	}
	if name == "" {
		r.Reporter.Errorf("--name is required")
		exitFn(1)
		return
	}
	handler := &hyperfleetNodePoolCreate{
		userOptions: userOptions,
		argv:        argv,
	}
	if err := handler.validateAndParseUserOptions(); err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
		return
	}

	clusterKey, err := ocm.GetClusterKey()
	if err != nil || clusterKey == "" {
		r.Reporter.Errorf("--cluster is required")
		exitFn(1)
		return
	}
	clusterUID, err := hyperfleet.ResolveClusterUID(context.Background(), r.HyperFleetClient, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
		return
	}
	handler.clusterKey = clusterKey
	handler.clusterUID = clusterUID
	if err := hfpathbind.RunCreateNodePool(
		context.Background(),
		r,
		nil,
		&hfNodePoolInput,
		handler,
		clusterUID,
	); err != nil {
		exitFn(1)
	}
}

// hyperfleetNodePoolCreate implements hfpathbind.NodePoolCreateHandler for rosa create machinepool.
type hyperfleetNodePoolCreate struct {
	hfpathbind.GeneratedNodePoolCreatePrompt
	userOptions          *mpOpts.CreateMachinepoolUserOptions
	argv                 []string
	clusterKey           string
	clusterUID           string
	userOptionsValidated bool
	labels               map[string]string
	taints               []hypershiftv1beta1.Taint
	resourceTags         []hypershiftv1beta1.AWSResourceTag
}

func (h *hyperfleetNodePoolCreate) validateAndParseUserOptions() error {
	if h.userOptionsValidated {
		return nil
	}
	if len(h.userOptions.Tags) > maxAWSResourceTags {
		return fmt.Errorf("%s", "Invalid machine pool AWS tags: Resource has too many AWS tags")
	}

	labels, err := parseHyperfleetLabels(h.userOptions.Labels)
	if err != nil {
		return err
	}

	taints, err := parseHyperfleetTaints(h.userOptions.Taints)
	if err != nil {
		return err
	}

	resourceTags, err := parseHyperfleetResourceTags(h.userOptions.Tags)
	if err != nil {
		return err
	}

	h.labels = labels
	h.taints = taints
	h.resourceTags = resourceTags
	h.userOptionsValidated = true
	return nil
}

func parseHyperfleetLabels(input string) (map[string]string, error) {
	if input != "" && input != `""` {
		entries := strings.Split(input, ",")
		for i, label := range entries {
			if label == "" && i == len(entries)-1 {
				continue
			}
			if strings.Count(label, "=") != 1 {
				return nil, fmt.Errorf("expected key=value format for labels")
			}
		}
	}
	return mpHelpers.ParseLabels(input)
}

func parseHyperfleetTaints(input string) ([]hypershiftv1beta1.Taint, error) {
	if input == "" || input == `""` {
		return nil, nil
	}

	entries := strings.Split(input, ",")
	trimmedEntries := make([]string, 0, len(entries))
	for i, taint := range entries {
		if taint == "" && i == len(entries)-1 && i > 0 {
			continue
		}
		originalTaint := taint
		taint = strings.TrimSpace(taint)
		if !strings.Contains(taint, "=") || !strings.Contains(taint, ":") {
			return nil, fmt.Errorf("expected key=value:scheduleType format for taints. Got '%s'", originalTaint)
		}
		if strings.Count(taint, "=") != 1 || strings.Count(taint, ":") != 1 {
			return nil, fmt.Errorf("invalid taint format: '%s'. Expected format is '<key>=<value>:<effect>'", taint)
		}
		trimmedEntries = append(trimmedEntries, taint)
	}

	parsedTaints, err := mpHelpers.ParseTaints(strings.Join(trimmedEntries, ","))
	if err != nil {
		return nil, err
	}

	taints := make([]hypershiftv1beta1.Taint, 0, len(parsedTaints))
	for _, taintBuilder := range parsedTaints {
		parsed, err := taintBuilder.Build()
		if err != nil {
			return nil, err
		}
		taints = append(taints, hypershiftv1beta1.Taint{
			Key:    parsed.Key(),
			Value:  parsed.Value(),
			Effect: corev1.TaintEffect(parsed.Effect()),
		})
	}
	return taints, nil
}

func parseHyperfleetResourceTags(input []string) ([]hypershiftv1beta1.AWSResourceTag, error) {
	if len(input) == 0 {
		return nil, nil
	}
	if err := rosaaws.UserTagValidator(input); err != nil {
		return nil, err
	}

	delimiter := rosaaws.GetTagsDelimiter(input)
	tags := make([]hypershiftv1beta1.AWSResourceTag, 0, len(input))
	for _, tag := range input {
		parts := strings.SplitN(tag, delimiter, 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tag format for tag '%s'. Expected tag format: 'key value'", tag)
		}
		tags = append(tags, hypershiftv1beta1.AWSResourceTag{
			Key:   strings.TrimSpace(parts[0]),
			Value: strings.TrimSpace(parts[1]),
		})
	}
	return tags, nil
}

func (h *hyperfleetNodePoolCreate) PreRequest(
	_ context.Context,
	r *rosa.Runtime,
	input *hfpathbind.NodePoolCreateInput,
) error {
	if err := h.validateAndParseUserOptions(); err != nil {
		return err
	}

	// Bridge: all nodepool flags (--name, --replicas, --instance-type, --subnet) share
	// names with OCM registrations so registerIfNew skips them; read from userOptions.
	// When these OCM flag registrations are removed, the bridges below can be dropped.
	name := h.userOptions.Name
	if name == "" && len(h.argv) > 0 {
		name = h.argv[0]
	}
	input.Name = name
	input.ClusterName = h.clusterKey

	instanceType := h.userOptions.InstanceType
	if instanceType == "" {
		instanceType = mpOpts.DefaultInstanceType
	}
	input.InstanceType = instanceType

	input.SubnetID = h.userOptions.Subnet

	replicas := int32(h.userOptions.Replicas)
	input.Replicas = &replicas
	if h.userOptions.AutoscalingEnabled {
		minReplicas := int32(h.userOptions.MinReplicas)
		maxReplicas := int32(h.userOptions.MaxReplicas)
		input.Min = &minReplicas
		input.Max = &maxReplicas
		input.Replicas = nil
	}

	if input.Name == "" {
		return fmt.Errorf("--name is required")
	}

	// Auto-select subnet from existing node pools if not provided (matches V1 behavior)
	if input.SubnetID == "" {
		ctx := context.Background()
		npList, err := r.HyperFleetClient.HyperfleetV1alpha1().NodePools(clusterNamespacePrefix+h.clusterUID).
			List(ctx, platform.ListOptions{})
		if err == nil && len(npList.Items) > 0 {
			// Use the first node pool's subnet as default
			if npList.Items[0].Spec.NodePool.Platform.AWS != nil && npList.Items[0].Spec.NodePool.Platform.AWS.Subnet.ID != nil {
				input.SubnetID = *npList.Items[0].Spec.NodePool.Platform.AWS.Subnet.ID
				r.Reporter.Debugf("Auto-selected subnet %s from existing machine pool", input.SubnetID)
			}
		}

		// Still require subnet if we couldn't auto-select
		if input.SubnetID == "" {
			return fmt.Errorf("--subnet is required for Platform API machine pool creation")
		}
	}
	return nil
}

func (h *hyperfleetNodePoolCreate) PostExpand(
	ctx context.Context,
	r *rosa.Runtime,
	_ *hfpathbind.NodePoolCreateInput,
	obj *v1alpha1.NodePool,
) error {
	if err := h.validateAndParseUserOptions(); err != nil {
		return err
	}

	cluster, err := r.HyperFleetClient.HyperfleetV1alpha1().Clusters().
		Get(ctx, h.clusterUID, platform.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get cluster %q: %w", h.clusterKey, err)
	}

	var rolesRef hypershiftv1beta1.AWSRolesRef
	if cluster.Spec.HostedCluster.Platform.AWS != nil {
		rolesRef = cluster.Spec.HostedCluster.Platform.AWS.RolesRef
	}
	instanceProfile := hyperfleet.InstanceProfileFromRolesRef(rolesRef)
	if instanceProfile == "" {
		return fmt.Errorf("cannot derive worker instance profile from cluster roles ref")
	}

	obj.Spec.NodePool.Platform.Type = hypershiftv1beta1.AWSPlatform

	// Ensure AWS platform is initialized
	if obj.Spec.NodePool.Platform.AWS == nil {
		obj.Spec.NodePool.Platform.AWS = &hypershiftv1beta1.AWSNodePoolPlatform{}
	}

	obj.Spec.NodePool.Platform.AWS.InstanceProfile = instanceProfile

	// Root disk size
	if h.userOptions.RootDiskSize != "" {
		if obj.Spec.NodePool.Platform.AWS.RootVolume != nil && obj.Spec.NodePool.Platform.AWS.RootVolume.Size != 0 {
			return fmt.Errorf("--disk-size and --size cannot be used together")
		}
		size, err := ocm.ParseDiskSizeToGigibyte(h.userOptions.RootDiskSize)
		if err != nil {
			return fmt.Errorf("invalid --disk-size: %w", err)
		}
		if obj.Spec.NodePool.Platform.AWS.RootVolume == nil {
			obj.Spec.NodePool.Platform.AWS.RootVolume = &hypershiftv1beta1.Volume{}
		}
		obj.Spec.NodePool.Platform.AWS.RootVolume.Size = int64(size)
	}

	// Spot instances
	if h.userOptions.UseSpotInstances {
		if obj.Spec.NodePool.Platform.AWS.Placement == nil {
			obj.Spec.NodePool.Platform.AWS.Placement = &hypershiftv1beta1.PlacementOptions{}
		}
		obj.Spec.NodePool.Platform.AWS.Placement.MarketType = hypershiftv1beta1.MarketTypeSpot
		if h.userOptions.SpotMaxPrice != "" {
			obj.Spec.NodePool.Platform.AWS.Placement.Spot.MaxPrice = h.userOptions.SpotMaxPrice
		}
	}

	// Security groups
	if len(h.userOptions.SecurityGroupIds) > 0 {
		obj.Spec.NodePool.Platform.AWS.SecurityGroups = make(
			[]hypershiftv1beta1.AWSResourceReference,
			len(h.userOptions.SecurityGroupIds),
		)
		for i, sgID := range h.userOptions.SecurityGroupIds {
			idCopy := sgID
			obj.Spec.NodePool.Platform.AWS.SecurityGroups[i] = hypershiftv1beta1.AWSResourceReference{
				ID: &idCopy,
			}
		}
	}

	// Resource tags
	if len(h.resourceTags) > 0 {
		obj.Spec.NodePool.Platform.AWS.ResourceTags = h.resourceTags
	}

	// Labels
	if len(h.labels) > 0 {
		obj.Spec.Labels = h.labels
	}

	// Taints
	if len(h.taints) > 0 {
		obj.Spec.NodePool.Taints = h.taints
	}

	// autoRepair is exposed as a mutable top-level Platform API field.
	autoRepair := h.userOptions.Autorepair
	obj.Spec.AutoRepair = &autoRepair

	return nil
}

func (h *hyperfleetNodePoolCreate) PostResponse(_ context.Context, r *rosa.Runtime, created *v1alpha1.NodePool) error {
	r.Reporter.Infof("Machine pool '%s' created successfully on hosted cluster '%s'", created.Name, h.clusterKey)
	return nil
}
