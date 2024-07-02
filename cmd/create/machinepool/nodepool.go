package machinepool

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/briandowns/spinner"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/helper/features"
	"github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/helper/versions"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/securitygroups"
	"github.com/openshift/rosa/pkg/machinepool"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

func addNodePool(cmd *cobra.Command, clusterKey string, cluster *cmv1.Cluster, r *rosa.Runtime) {
	var err error

	isAvailabilityZoneSet := cmd.Flags().Changed("availability-zone")
	isSubnetSet := cmd.Flags().Changed("subnet")
	if isSubnetSet && isAvailabilityZoneSet {
		r.Reporter.Errorf("Setting both `subnet` and `availability-zone` flag is not supported." +
			" Please select `subnet` or `availability-zone` to create a single availability zone machine pool")
		os.Exit(1)
	}

	isMultiAvailabilityZoneSet := cmd.Flags().Changed("multi-availability-zone")
	if isMultiAvailabilityZoneSet {
		r.Reporter.Errorf("Setting the `multi-availability-zone` flag is not allowed for Hosted Control Plane clusters")
		os.Exit(1)
	}

	// Machine pool name:
	name := strings.Trim(args.name, " \t")
	if name == "" && !interactive.Enabled() {
		interactive.Enable()
		r.Reporter.Infof("Enabling interactive mode")
	}
	if name == "" || interactive.Enabled() {
		name, err = interactive.GetString(interactive.Input{
			Question: "Machine pool name",
			Default:  name,
			Required: true,
			Validators: []interactive.Validator{
				interactive.RegExp(machinePoolKeyRE.String()),
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid name for the machine pool: %s", err)
			os.Exit(1)
		}
	}
	name = strings.Trim(name, " \t")
	if !machinePoolKeyRE.MatchString(name) {
		r.Reporter.Errorf("Expected a valid name for the machine pool")
		os.Exit(1)
	}

	// OpenShift version:
	isVersionSet := cmd.Flags().Changed("version")
	version := args.version
	if isVersionSet || interactive.Enabled() {
		// NodePool will take channel group from the cluster
		channelGroup := cluster.Version().ChannelGroup()
		clusterVersion := cluster.Version().RawID()
		// This is called in HyperShift, but we don't want to exclude version which are HCP disabled for node pools
		// so we pass the relative parameter as false
		_, versionList, err := versions.GetVersionList(r, channelGroup, true, true, false, false)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		// Calculate the minimal version for a new hosted machine pool
		minVersion, err := versions.GetMinimalHostedMachinePoolVersion(clusterVersion)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		// Filter the available list of versions for a hosted machine pool
		filteredVersionList := versions.GetFilteredVersionList(versionList, minVersion, clusterVersion)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		if version == "" {
			version = clusterVersion
		}
		if interactive.Enabled() {
			version, err = interactive.GetOption(interactive.Input{
				Question: "OpenShift version",
				Help:     cmd.Flags().Lookup("version").Usage,
				Options:  filteredVersionList,
				Default:  version,
				Required: true,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid OpenShift version: %s", err)
				os.Exit(1)
			}
		}
		// This is called in HyperShift, but we don't want to exclude version which are HCP disabled for node pools
		// so we pass the relative parameter as false
		version, err = r.OCMClient.ValidateVersion(version, filteredVersionList, channelGroup, true, false)
		if err != nil {
			r.Reporter.Errorf("Expected a valid OpenShift version: %s", err)
			os.Exit(1)
		}
	}

	// Allow the user to select subnet for a single AZ BYOVPC cluster
	subnet := getSubnetFromUser(cmd, r, isSubnetSet, cluster)

	// Select availability zone if the user didn't select subnet
	if subnet == "" {
		subnet, err = getSubnetFromAvailabilityZone(cmd, r, isAvailabilityZoneSet, cluster)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	isMinReplicasSet := cmd.Flags().Changed("min-replicas")
	isMaxReplicasSet := cmd.Flags().Changed("max-replicas")
	isAutoscalingSet := cmd.Flags().Changed("enable-autoscaling")
	isReplicasSet := cmd.Flags().Changed("replicas")

	minReplicas := args.minReplicas
	maxReplicas := args.maxReplicas
	autoscaling := args.autoscalingEnabled
	replicas := args.replicas

	// Autoscaling
	if !isReplicasSet && !autoscaling && !isAutoscalingSet && interactive.Enabled() {
		autoscaling, err = interactive.GetBool(interactive.Input{
			Question: "Enable autoscaling",
			Help:     cmd.Flags().Lookup("enable-autoscaling").Usage,
			Default:  autoscaling,
			Required: false,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value for enable-autoscaling: %s", err)
			os.Exit(1)
		}
	}

	// TODO Update the autoscaling input validator when multi-AZ is implemented
	if autoscaling {
		// if the user set replicas and enabled autoscaling
		if isReplicasSet {
			r.Reporter.Errorf("Replicas can't be set when autoscaling is enabled")
			os.Exit(1)
		}
		if interactive.Enabled() || !isMinReplicasSet {
			minReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Min replicas",
				Help:     cmd.Flags().Lookup("min-replicas").Usage,
				Default:  minReplicas,
				Required: true,
				Validators: []interactive.Validator{
					machinepools.MinNodePoolReplicaValidator(true),
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid number of min replicas: %s", err)
				os.Exit(1)
			}
		}
		err = machinepools.MinNodePoolReplicaValidator(true)(minReplicas)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		if interactive.Enabled() || !isMaxReplicasSet {
			maxReplicas, err = interactive.GetInt(interactive.Input{
				Question: "Max replicas",
				Help:     cmd.Flags().Lookup("max-replicas").Usage,
				Default:  maxReplicas,
				Required: true,
				Validators: []interactive.Validator{
					machinepools.MaxNodePoolReplicaValidator(minReplicas),
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid number of max replicas: %s", err)
				os.Exit(1)
			}
		}
		err = machinepools.MaxNodePoolReplicaValidator(minReplicas)(maxReplicas)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	} else {
		// if the user set min/max replicas and hasn't enabled autoscaling
		if isMinReplicasSet || isMaxReplicasSet {
			r.Reporter.Errorf("Autoscaling must be enabled in order to set min and max replicas")
			os.Exit(1)
		}
		if interactive.Enabled() || !isReplicasSet {
			replicas, err = interactive.GetInt(interactive.Input{
				Question: "Replicas",
				Help:     cmd.Flags().Lookup("replicas").Usage,
				Default:  replicas,
				Required: true,
				Validators: []interactive.Validator{
					machinepools.MinNodePoolReplicaValidator(false),
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid number of replicas: %s", err)
				os.Exit(1)
			}
		}
		err = machinepools.MinNodePoolReplicaValidator(false)(replicas)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	existingLabels := make(map[string]string, 0)
	labelMap := machinepools.GetLabelMap(cmd, r, existingLabels, args.labels)

	existingTaints := make([]*cmv1.Taint, 0)
	taintBuilders := machinepools.GetTaints(cmd, r, existingTaints, args.taints)

	isSecurityGroupIdsSet := cmd.Flags().Changed(securitygroups.MachinePoolSecurityGroupFlag)
	securityGroupIds := args.securityGroupIds
	isVersionCompatibleSecurityGroupIds, err := features.IsFeatureSupported(
		features.AdditionalDay2SecurityGroupsHcpFeature, version)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	if interactive.Enabled() && !isSecurityGroupIdsSet && isVersionCompatibleSecurityGroupIds {
		securityGroupIds, err = getSecurityGroupsOption(r, cmd, cluster)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}
	for i, sg := range securityGroupIds {
		securityGroupIds[i] = strings.TrimSpace(sg)
	}

	awsTags := machinepools.GetAwsTags(cmd, r, args.tags)

	npBuilder := cmv1.NewNodePool()
	npBuilder.ID(name).Labels(labelMap).
		Taints(taintBuilders...)

	if autoscaling {
		npBuilder = npBuilder.Autoscaling(
			cmv1.NewNodePoolAutoscaling().
				MinReplica(minReplicas).
				MaxReplica(maxReplicas))
	} else {
		npBuilder = npBuilder.Replicas(replicas)
	}

	if subnet != "" {
		npBuilder.Subnet(subnet)
	}

	// Machine pool instance type:
	// NodePools don't support MultiAZ yet, so the availabilityZonesFilters is calculated from the cluster

	// Machine pool instance type:
	instanceType := args.instanceType
	if instanceType == "" && !interactive.Enabled() {
		r.Reporter.Errorf("You must supply a valid instance type")
		os.Exit(1)
	}

	var spin *spinner.Spinner
	if r.Reporter.IsTerminal() && !output.HasFlag() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		r.Reporter.Infof("Checking available instance types for machine pool '%s'", name)
		spin.Start()
	}

	availabilityZonesFilter := cluster.Nodes().AvailabilityZones()

	// If the user selects a subnet which is in a different AZ than day 1, the instance type list should be filter
	// by the new AZ not the cluster ones
	if subnet != "" {
		availabilityZone, err := r.AWSClient.GetSubnetAvailabilityZone(subnet)
		if err != nil {
			r.Reporter.Errorf(fmt.Sprintf("%s", err))
			os.Exit(1)
		}
		availabilityZonesFilter = []string{availabilityZone}
	}

	instanceTypeList, err := r.OCMClient.GetAvailableMachineTypesInRegion(cluster.Region().ID(),
		availabilityZonesFilter, cluster.AWS().STS().RoleARN(), r.AWSClient)
	if err != nil {
		r.Reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}

	if spin != nil {
		spin.Stop()
	}

	if interactive.Enabled() {
		if instanceType == "" {
			instanceType = instanceTypeList.Items[0].MachineType.ID()
		}
		instanceType, err = interactive.GetOption(interactive.Input{
			Question: "Instance type",
			Help:     cmd.Flags().Lookup("instance-type").Usage,
			Options:  instanceTypeList.GetAvailableIDs(cluster.MultiAZ()),
			Default:  instanceType,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid instance type: %s", err)
			os.Exit(1)
		}
	}

	err = instanceTypeList.ValidateMachineType(instanceType, cluster.MultiAZ())
	if err != nil {
		r.Reporter.Errorf("Expected a valid instance type: %s", err)
		os.Exit(1)
	}

	autorepair := args.autorepair
	if interactive.Enabled() {
		autorepair, err = interactive.GetBool(interactive.Input{
			Question: "Autorepair",
			Help:     cmd.Flags().Lookup("autorepair").Usage,
			Default:  autorepair,
			Required: false,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value for autorepair: %s", err)
			os.Exit(1)
		}
	}

	npBuilder.AutoRepair(autorepair)

	var inputTuningConfig []string
	tuningConfigs := args.tuningConfigs
	// Get the list of available tuning configs
	availableTuningConfigs, err := r.OCMClient.GetTuningConfigsName(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	if tuningConfigs != "" {
		if len(availableTuningConfigs) > 0 {
			inputTuningConfig = strings.Split(tuningConfigs, ",")
		} else {
			// Parameter will be ignored
			r.Reporter.Warnf("No tuning config available for cluster '%s'. "+
				"Any tuning config in input will be ignored", cluster.ID())
		}
	}
	if interactive.Enabled() {
		// Skip if no tuning configs are available
		if len(availableTuningConfigs) > 0 {
			inputTuningConfig, err = interactive.GetMultipleOptions(interactive.Input{
				Question: "Tuning configs",
				Help:     cmd.Flags().Lookup("tuning-configs").Usage,
				Options:  availableTuningConfigs,
				Default:  inputTuningConfig,
				Required: false,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid value for tuning configs: %s", err)
				os.Exit(1)
			}
		}
	}

	if len(inputTuningConfig) != 0 {
		npBuilder.TuningConfigs(inputTuningConfig...)
	}

	kubeletConfigs := args.kubeletConfigs

	if kubeletConfigs != "" || interactive.Enabled() {
		var inputKubeletConfigs []string
		// Get the list of available kubelet configs
		availableKubeletConfigs, err := r.OCMClient.ListKubeletConfigNames(cluster.ID())
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		if len(availableKubeletConfigs) > 0 {
			inputKubeletConfigs = strings.Split(kubeletConfigs, ",")
		} else {
			// Parameter will be ignored
			r.Reporter.Warnf("No kubelet configs available for cluster '%s'. "+
				"Any kubelet config in input will be ignored", cluster.ID())
		}

		if interactive.Enabled() {
			// Skip if no kubelet configs are available
			if len(availableKubeletConfigs) > 0 {
				inputKubeletConfigs, err = interactive.GetMultipleOptions(interactive.Input{
					Question: "Kubelet config",
					Help:     cmd.Flags().Lookup("kubelet-configs").Usage,
					Options:  availableKubeletConfigs,
					Default:  inputKubeletConfigs,
					Required: false,
					Validators: []interactive.Validator{
						machinepool.ValidateKubeletConfig,
					},
				})
				if err != nil {
					r.Reporter.Errorf("Expected a valid value for kubelet config: %s", err)
					os.Exit(1)
				}
			}
		}

		err = machinepool.ValidateKubeletConfig(inputKubeletConfigs)
		if err != nil {
			r.Reporter.Errorf(err.Error())
			os.Exit(1)
		}

		if len(inputKubeletConfigs) != 0 {
			npBuilder.KubeletConfigs(inputKubeletConfigs...)
		}
	}

	npBuilder.AWSNodePool(createAwsNodePoolBuilder(instanceType, securityGroupIds, awsTags))

	nodeDrainGracePeriod := args.nodeDrainGracePeriod
	if interactive.Enabled() {
		nodeDrainGracePeriod, err = interactive.GetString(interactive.Input{
			Question: "Node drain grace period",
			Help:     cmd.Flags().Lookup("node-drain-grace-period").Usage,
			Default:  nodeDrainGracePeriod,
			Required: false,
			Validators: []interactive.Validator{
				machinepools.ValidateNodeDrainGracePeriod,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value for Node drain grace period: %s", err)
			os.Exit(1)
		}
	}
	if nodeDrainGracePeriod != "" {
		nodeDrainBuilder, err := machinepools.CreateNodeDrainGracePeriodBuilder(nodeDrainGracePeriod)
		if err != nil {
			r.Reporter.Errorf(err.Error())
			os.Exit(1)
		}
		npBuilder.NodeDrainGracePeriod(nodeDrainBuilder)
	}

	isMaxSurgeSet := cmd.Flags().Changed("max-surge")
	isMaxUnavailableSet := cmd.Flags().Changed("max-unavailable")
	if isMaxSurgeSet && isMaxUnavailableSet {
		maxSurge := args.maxSurge
		if interactive.Enabled() {
			maxSurge, err = interactive.GetString(interactive.Input{
				Question: "Max surge",
				Help:     cmd.Flags().Lookup("max-surge").Usage,
				Default:  maxSurge,
				Required: false,
				Validators: []interactive.Validator{
					machinepools.ValidateUpgradeMaxSurgeUnavailable,
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid value for max surge: %s", err)
				os.Exit(1)
			}
		}

		maxUnavailable := args.maxUnavailable
		if interactive.Enabled() {
			maxUnavailable, err = interactive.GetString(interactive.Input{
				Question: "Max unavailable",
				Help:     cmd.Flags().Lookup("max-unavailable").Usage,
				Default:  maxUnavailable,
				Required: false,
				Validators: []interactive.Validator{
					machinepools.ValidateUpgradeMaxSurgeUnavailable,
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid value for max unavailable: %s", err)
				os.Exit(1)
			}
		}
		if maxSurge != "" || maxUnavailable != "" {
			mgmtUpgradeBuilder := cmv1.NewNodePoolManagementUpgrade()
			if maxSurge != "" {
				mgmtUpgradeBuilder.MaxSurge(maxSurge)
			}
			if maxUnavailable != "" {
				mgmtUpgradeBuilder.MaxUnavailable(maxUnavailable)
			}
			npBuilder.ManagementUpgrade(mgmtUpgradeBuilder)
		}
	}

	if version != "" {
		npBuilder.Version(cmv1.NewVersion().ID(version))
	}

	nodePool, err := npBuilder.Build()
	if err != nil {
		r.Reporter.Errorf("Failed to create machine pool for hosted cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	createdNodePool, err := r.OCMClient.CreateNodePool(cluster.ID(), nodePool)
	if err != nil {
		r.Reporter.Errorf("Failed to add machine pool to hosted cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if output.HasFlag() {
		if err = output.Print(createdNodePool); err != nil {
			r.Reporter.Errorf("Unable to print machine pool: %v", err)
			os.Exit(1)
		}
	} else {
		r.Reporter.Infof("Machine pool '%s' created successfully on hosted cluster '%s'", createdNodePool.ID(), clusterKey)
		r.Reporter.Infof("To view the machine pool details, run 'rosa describe machinepool --cluster %s --machinepool %s'",
			clusterKey, name)
		r.Reporter.Infof("To view all machine pools, run 'rosa list machinepools --cluster %s'", clusterKey)
	}
}

func getSubnetFromAvailabilityZone(cmd *cobra.Command, r *rosa.Runtime, isAvailabilityZoneSet bool,
	cluster *cmv1.Cluster) (string, error) {

	privateSubnets, err := r.AWSClient.GetVPCPrivateSubnets(cluster.AWS().SubnetIDs()[0])
	if err != nil {
		return "", err
	}

	// Fetching the availability zones from the VPC private subnets
	subnetsMap := make(map[string][]types.Subnet)
	for _, privateSubnet := range privateSubnets {
		subnetsPerAZ, exist := subnetsMap[*privateSubnet.AvailabilityZone]
		if !exist {
			subnetsPerAZ = []types.Subnet{privateSubnet}
		} else {
			subnetsPerAZ = append(subnetsPerAZ, privateSubnet)
		}
		subnetsMap[*privateSubnet.AvailabilityZone] = subnetsPerAZ
	}
	availabilityZones := make([]string, 0)
	for availabilizyZone := range subnetsMap {
		availabilityZones = append(availabilityZones, availabilizyZone)
	}

	availabilityZone := cluster.Nodes().AvailabilityZones()[0]
	if !isAvailabilityZoneSet && interactive.Enabled() {
		availabilityZone, err = interactive.GetOption(interactive.Input{
			Question: "AWS availability zone",
			Help:     cmd.Flags().Lookup("availability-zone").Usage,
			Options:  availabilityZones,
			Default:  availabilityZone,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid AWS availability zone: %s", err)
			os.Exit(1)
		}
	} else if isAvailabilityZoneSet {
		availabilityZone = args.availabilityZone
	}

	if subnets, ok := subnetsMap[availabilityZone]; ok {
		if len(subnets) == 1 {
			return *subnets[0].SubnetId, nil
		}
		r.Reporter.Infof("There are several subnets for availability zone '%s'", availabilityZone)
		interactive.Enable()

		// should only get the subnets from the selected availability zone
		subnetOptions := make([]string, 0)
		for _, subnet := range subnets {
			subnetOptions = append(subnetOptions, aws.SetSubnetOption(subnet))
		}

		subnetOption, err := interactive.GetOption(interactive.Input{
			Question: "Subnet ID",
			Help:     cmd.Flags().Lookup("subnet").Usage,
			Options:  subnetOptions,
			Default:  subnetOptions[0],
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid AWS subnet: %s", err)
			os.Exit(1)
		}
		subnet := aws.ParseOption(subnetOption)
		return subnet, nil
	}

	return "", fmt.Errorf("Failed to find a private subnet for '%s' availability zone", availabilityZone)
}

func createAwsNodePoolBuilder(
	instanceType string,
	securityGroupIds []string,
	awsTags map[string]string,
) *cmv1.AWSNodePoolBuilder {
	awsNpBuilder := cmv1.NewAWSNodePool().InstanceType(instanceType)

	if len(securityGroupIds) > 0 {
		awsNpBuilder.AdditionalSecurityGroupIds(securityGroupIds...)
	}

	if len(awsTags) > 0 {
		awsNpBuilder.Tags(awsTags)
	}

	return awsNpBuilder
}
