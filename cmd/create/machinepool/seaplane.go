// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	rosaaws "github.com/openshift/rosa/pkg/aws"
	awsapi "github.com/openshift/rosa/pkg/aws/api_interface"
	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
	mpValidation "github.com/openshift/rosa/pkg/machinepool"
	"github.com/openshift/rosa/pkg/ocm"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/seaplane"
	seaplanepathbind "github.com/openshift/rosa/pkg/seaplane/pathbind"
)

const maxAWSResourceTags = 25

// seaplaneNodePoolInput is the backing store for seaplane-specific create machinepool flags.
var seaplaneNodePoolInput seaplanepathbind.NodePoolCreateInput

var (
	newNodePoolEC2Client = func(config awssdk.Config) awsapi.Ec2ApiClient { return ec2.NewFromConfig(config) }

	exitFn = func(code int) { os.Exit(code) }

	seaplaneCreateMachinePool = func(userOptions *mpOpts.CreateMachinepoolUserOptions, argv []string, cmd *cobra.Command) {
		r := rosa.NewRuntime().WithSeaplane()
		defer r.Cleanup()

		handler := &seaplaneNodePoolCreate{
			userOptions: userOptions,
			argv:        argv,
			cmd:         cmd,
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
		cluster, err := seaplane.ResolveCluster(context.Background(), r.SeaplaneClient, clusterKey)
		if err != nil {
			r.Reporter.Errorf("%v", err)
			exitFn(1)
			return
		}

		handler.clusterKey = clusterKey
		handler.clusterName = cluster.Name
		handler.clusterUID = string(cluster.UID)
		if err := seaplanepathbind.RunCreateNodePool(
			context.Background(),
			r,
			cmd,
			&seaplaneNodePoolInput,
			handler,
		); err != nil {
			r.Reporter.Errorf("%v", err)
			exitFn(1)
		}
	}
)

// runSeaplaneCreate is a thin wrapper for direct test invocation without a real cobra.Command.
func runSeaplaneCreate(r *rosa.Runtime, userOptions *mpOpts.CreateMachinepoolUserOptions, argv []string) {
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
	handler := &seaplaneNodePoolCreate{
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
	cluster, err := seaplane.ResolveCluster(context.Background(), r.SeaplaneClient, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
		return
	}
	handler.clusterKey = clusterKey
	handler.clusterName = cluster.Name
	handler.clusterUID = string(cluster.UID)
	if err := seaplanepathbind.RunCreateNodePool(
		context.Background(),
		r,
		nil,
		&seaplaneNodePoolInput,
		handler,
	); err != nil {
		exitFn(1)
	}
}

// seaplaneNodePoolCreate implements seaplanepathbind.NodePoolCreateHandler for rosa create machinepool.
type seaplaneNodePoolCreate struct {
	seaplanepathbind.GeneratedNodePoolCreatePrompt
	userOptions          *mpOpts.CreateMachinepoolUserOptions
	argv                 []string
	cmd                  *cobra.Command
	clusterKey           string
	clusterName          string
	clusterUID           string
	userOptionsValidated bool
	labels               map[string]string
	taints               []hypershiftv1beta1.Taint
	tags                 map[string]string
}

func (h *seaplaneNodePoolCreate) validateAndParseUserOptions() error {
	if h.userOptionsValidated {
		return nil
	}
	if h.cmd != nil && h.cmd.Flags().Changed("multi-availability-zone") {
		return fmt.Errorf("setting `multi-availability-zone` flag is not supported for HCP clusters")
	}
	if h.cmd != nil && h.cmd.Flags().Changed("version") && h.userOptions.Version == "" {
		return fmt.Errorf("expected a valid OpenShift version: --version requires a non-empty value")
	}
	httpTokensFlagChanged := h.cmd != nil && h.cmd.Flags().Lookup("ec2-metadata-http-tokens") != nil &&
		h.cmd.Flags().Changed("ec2-metadata-http-tokens")
	if h.userOptions.EC2MetadataHttpTokens == "" && httpTokensFlagChanged {
		//nolint:lll // Preserve the CLI validation message.
		return fmt.Errorf("expected a valid http tokens value : ec2-metadata-http-tokens value should be one of 'required', 'optional'")
	}
	if h.userOptions.EC2MetadataHttpTokens != "" {
		if err := ocm.ValidateHttpTokensValue(h.userOptions.EC2MetadataHttpTokens); err != nil {
			return fmt.Errorf("expected a valid http tokens value : %v", err)
		}
	}
	name := h.userOptions.Name
	if name == "" && len(h.argv) > 0 {
		name = h.argv[0]
	}
	if name != "" {
		if err := mpValidation.ValidateMachinePoolName(name); err != nil {
			return err
		}
	}
	if err := h.validateReplicas(); err != nil {
		return err
	}
	labels, err := mpHelpers.ParseSeaplaneLabels(h.userOptions.Labels)
	if err != nil {
		return err
	}

	taints, err := mpHelpers.ParseSeaplaneTaints(h.userOptions.Taints)
	if err != nil {
		return err
	}

	var tags map[string]string
	if len(h.userOptions.Tags) > 0 {
		tags, err = rosaaws.ParseUserTags(h.userOptions.Tags, rosaaws.UserTagOptions{})
		if err != nil {
			return fmt.Errorf("invalid machine pool AWS tags: %w", err)
		}
	}
	if len(tags) > maxAWSResourceTags {
		return fmt.Errorf("%s", "Invalid machine pool AWS tags: Resource has too many AWS tags")
	}

	h.labels = labels
	h.taints = taints
	h.tags = tags
	h.userOptionsValidated = true
	return nil
}

func (h *seaplaneNodePoolCreate) validateReplicas() error {
	options := h.userOptions
	validation := mpValidation.ReplicaSizeValidation{
		IsHostedCp:  true,
		Autoscaling: options.AutoscalingEnabled,
		MinReplicas: options.MinReplicas,
	}
	if options.AutoscalingEnabled {
		if h.cmd != nil && h.cmd.Flags().Changed("replicas") {
			return fmt.Errorf("replicas can't be set when autoscaling is enabled")
		}
		if err := validation.MinReplicaValidator()(options.MinReplicas); err != nil {
			return err
		}
		return validation.MaxReplicaValidator()(options.MaxReplicas)
	}
	if h.cmd != nil && (h.cmd.Flags().Changed("min-replicas") || h.cmd.Flags().Changed("max-replicas")) {
		return fmt.Errorf("autoscaling must be enabled in order to set min and max replicas")
	}
	return validation.MinReplicaValidator()(options.Replicas)
}

func (h *seaplaneNodePoolCreate) PreRequest(
	_ context.Context,
	r *rosa.Runtime,
	input *seaplanepathbind.NodePoolCreateInput,
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
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	input.Name = h.clusterName + "." + name
	input.ClusterName = h.clusterName

	instanceType := h.userOptions.InstanceType
	if instanceType == "" {
		instanceType = mpOpts.DefaultInstanceType
	}
	input.InstanceType = instanceType

	input.SubnetID = h.userOptions.Subnet
	input.EC2InstanceMetadataHttpTokens = h.userOptions.EC2MetadataHttpTokens
	if h.tags != nil {
		encodedTags, err := json.Marshal(h.tags)
		if err != nil {
			return fmt.Errorf("failed to encode --tags: %w", err)
		}
		input.Tags = string(encodedTags)
	}

	replicas := int32(h.userOptions.Replicas)
	input.Replicas = &replicas
	if h.userOptions.AutoscalingEnabled {
		minReplicas := int32(h.userOptions.MinReplicas)
		maxReplicas := int32(h.userOptions.MaxReplicas)
		input.Min = &minReplicas
		input.Max = &maxReplicas
		input.Replicas = nil
	}

	if h.userOptions.RootDiskSize != "" {
		size, err := ocm.ParseDiskSizeToGigibyte(h.userOptions.RootDiskSize)
		if err != nil {
			return fmt.Errorf("invalid --disk-size: %w", err)
		}
		sizeInt64 := int64(size)
		input.Size = &sizeInt64
	}

	// Auto-select subnet from existing node pools if not provided (matches V1 behavior)
	if input.SubnetID == "" {
		ctx := context.Background()
		npList, err := r.SeaplaneClient.SeaplaneV1alpha1().NodePools().
			List(ctx, platform.NodePoolListOptions{ClusterUID: h.clusterUID})
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

func (h *seaplaneNodePoolCreate) PostExpand(
	ctx context.Context,
	r *rosa.Runtime,
	_ *seaplanepathbind.NodePoolCreateInput,
	obj *v1alpha1.NodePool,
) error {
	if err := h.validateAndParseUserOptions(); err != nil {
		return err
	}

	cluster, err := r.SeaplaneClient.SeaplaneV1alpha1().Clusters().
		Get(ctx, h.clusterName, platform.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get cluster %q: %w", h.clusterKey, err)
	}
	if h.userOptions.Version != "" {
		image, err := seaplane.ResolveNodePoolReleaseImage(cluster, h.userOptions.Version,
			obj.Spec.NodePool.Release.Image)
		if err != nil {
			return err
		}
		obj.Spec.NodePool.Release.Image = image
	}

	var rolesRef hypershiftv1beta1.AWSRolesRef
	if cluster.Spec.HostedCluster.Platform.AWS != nil {
		rolesRef = cluster.Spec.HostedCluster.Platform.AWS.RolesRef
	}
	instanceProfile := seaplane.InstanceProfileFromRolesRef(rolesRef)
	if instanceProfile == "" {
		return fmt.Errorf("cannot derive worker instance profile from cluster roles ref")
	}

	obj.Spec.NodePool.Platform.Type = hypershiftv1beta1.AWSPlatform

	// Ensure AWS platform is initialized
	if obj.Spec.NodePool.Platform.AWS == nil {
		obj.Spec.NodePool.Platform.AWS = &v1alpha1.AWSNodePoolPlatform{}
	}

	obj.Spec.NodePool.Platform.AWS.InstanceProfile = instanceProfile

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

	// Customer tags belong on the NodePool's root-level tags field.
	if h.tags != nil {
		obj.Spec.Tags = h.tags
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

	if cluster.Spec.HostedCluster.Platform.AWS == nil ||
		cluster.Spec.HostedCluster.Platform.AWS.CloudProviderConfig == nil {
		return fmt.Errorf("cluster %q has no AWS platform configuration", h.clusterKey)
	}
	return rosaaws.ValidateMachinePoolAWS(ctx, newNodePoolEC2Client(r.AWSConfig), rosaaws.ValidateMachinePoolAWSRequest{
		SubnetID:     awssdk.ToString(obj.Spec.NodePool.Platform.AWS.Subnet.ID),
		VPCID:        cluster.Spec.HostedCluster.Platform.AWS.CloudProviderConfig.VPC,
		InstanceType: obj.Spec.NodePool.Platform.AWS.InstanceType,
	})
}

func (h *seaplaneNodePoolCreate) PostResponse(_ context.Context, r *rosa.Runtime, created *v1alpha1.NodePool) error {
	r.Reporter.Infof("Machine pool '%s' created successfully on hosted cluster '%s'",
		seaplane.NodePoolDisplayName(h.clusterName, created.Name), h.clusterKey)
	return nil
}
