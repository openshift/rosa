package machinepool

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/validation"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

func addMachinePool(cmd *cobra.Command, clusterKey string, cluster *cmv1.Cluster, r *rosa.Runtime) {
	var err error

	// Validate flags that are only allowed for multi-AZ clusters
	isMultiAvailabilityZoneSet := cmd.Flags().Changed("multi-availability-zone")
	if isMultiAvailabilityZoneSet && !cluster.MultiAZ() {
		r.Reporter.Errorf("Setting the `multi-availability-zone` flag is only allowed for multi-AZ clusters")
		os.Exit(1)
	}
	isAvailabilityZoneSet := cmd.Flags().Changed("availability-zone")
	if isAvailabilityZoneSet && !cluster.MultiAZ() {
		r.Reporter.Errorf("Setting the `availability-zone` flag is only allowed for multi-AZ clusters")
		os.Exit(1)
	}

	// Validate flags that are only allowed for BYOVPC cluster
	isSubnetSet := cmd.Flags().Changed("subnet")
	if !isBYOVPC(cluster) && isSubnetSet {
		r.Reporter.Errorf("Setting the `subnet` flag is only allowed for BYOVPC clusters")
		os.Exit(1)
	}

	if isSubnetSet && isAvailabilityZoneSet {
		r.Reporter.Errorf("Setting both `subnet` and `availability-zone` flag is not supported." +
			" Please select `subnet` or `availability-zone` to create a single availability zone machine pool")
		os.Exit(1)
	}

	// Validate `subnet` or `availability-zone` flags are set for a single AZ machine pool
	if isAvailabilityZoneSet && isMultiAvailabilityZoneSet && args.multiAvailabilityZone {
		r.Reporter.Errorf("Setting the `availability-zone` flag is only supported for creating a single AZ " +
			"machine pool in a multi-AZ cluster")
		os.Exit(1)
	}
	if isSubnetSet && isMultiAvailabilityZoneSet && args.multiAvailabilityZone {
		r.Reporter.Errorf("Setting the `subnet` flag is only supported for creating a single AZ machine pool")
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

	// Allow the user to select subnet for a single AZ BYOVPC cluster
	var subnet string
	if !cluster.MultiAZ() && isBYOVPC(cluster) {
		subnet = getSubnetFromUser(cmd, r, isSubnetSet, cluster)
	}

	// Single AZ machine pool for a multi-AZ cluster
	var multiAZMachinePool bool
	var availabilityZone string
	if cluster.MultiAZ() {
		// Choosing a single AZ machine pool implicitly (providing availability zone or subnet)
		if isAvailabilityZoneSet || isSubnetSet {
			isMultiAvailabilityZoneSet = true
			args.multiAvailabilityZone = false
		}

		if !isMultiAvailabilityZoneSet && interactive.Enabled() && !confirm.Yes() {
			multiAZMachinePool, err = interactive.GetBool(interactive.Input{
				Question: "Create multi-AZ machine pool",
				Help:     cmd.Flags().Lookup("multi-availability-zone").Usage,
				Default:  true,
				Required: false,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid value for create multi-AZ machine pool")
				os.Exit(1)
			}
		} else {
			multiAZMachinePool = args.multiAvailabilityZone
		}

		if !multiAZMachinePool {
			// Allow to create a single AZ machine pool providing the subnet
			if isBYOVPC(cluster) {
				subnet = getSubnetFromUser(cmd, r, isSubnetSet, cluster)
			}

			// Select availability zone if the user didn't select subnet
			if subnet == "" {
				availabilityZone = cluster.Nodes().AvailabilityZones()[0]
				if !isAvailabilityZoneSet && interactive.Enabled() {
					availabilityZone, err = interactive.GetOption(interactive.Input{
						Question: "AWS availability zone",
						Help:     cmd.Flags().Lookup("availability-zone").Usage,
						Options:  cluster.Nodes().AvailabilityZones(),
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

				if !helper.Contains(cluster.Nodes().AvailabilityZones(), availabilityZone) {
					r.Reporter.Errorf("Availability zone '%s' doesn't belong to the cluster's availability zones",
						availabilityZone)
					os.Exit(1)
				}
			}
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
					minReplicaValidator(multiAZMachinePool),
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid number of min replicas: %s", err)
				os.Exit(1)
			}
		}
		err = minReplicaValidator(multiAZMachinePool)(minReplicas)
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
					maxReplicaValidator(minReplicas, multiAZMachinePool),
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid number of max replicas: %s", err)
				os.Exit(1)
			}
		}
		err = maxReplicaValidator(minReplicas, multiAZMachinePool)(maxReplicas)
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
					minReplicaValidator(multiAZMachinePool),
				},
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid number of replicas: %s", err)
				os.Exit(1)
			}
		}
		err = minReplicaValidator(multiAZMachinePool)(replicas)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	var spin *spinner.Spinner
	if r.Reporter.IsTerminal() && !output.HasFlag() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		r.Reporter.Infof("Fetching instance types")
		spin.Start()
	}

	// Determine machine pool availability zones to filter supported machine types
	availabilityZonesFilter, err := getMachinePoolAvailabilityZones(r, cluster, multiAZMachinePool, availabilityZone,
		subnet)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Machine pool instance type:
	instanceType := args.instanceType
	instanceTypeList, err := r.OCMClient.GetAvailableMachineTypesInRegion(cluster.Region().ID(), availabilityZonesFilter,
		cluster.AWS().STS().RoleARN(), r.AWSClient)
	if err != nil {
		r.Reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}

	if spin != nil {
		spin.Stop()
	}

	if interactive.Enabled() {
		if instanceType == "" {
			instanceType = instanceTypeList[0].MachineType.ID()
		}
		instanceType, err = interactive.GetOption(interactive.Input{
			Question: "Instance type",
			Help:     cmd.Flags().Lookup("instance-type").Usage,
			Options:  instanceTypeList.GetAvailableIDs(cluster.MultiAZ()),
			Default:  instanceType,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid machine type: %s", err)
			os.Exit(1)
		}
	}
	if instanceType == "" {
		r.Reporter.Errorf("Expected a valid machine type")
		os.Exit(1)
	}
	err = instanceTypeList.ValidateMachineType(instanceType, cluster.MultiAZ())
	if err != nil {
		r.Reporter.Errorf("Expected a valid machine type: %s", err)
		os.Exit(1)
	}

	labelMap := getLabelMap(cmd, r)

	taintBuilders := getTaints(cmd, r)

	// Spot instances
	isSpotSet := cmd.Flags().Changed("use-spot-instances")
	isSpotMaxPriceSet := cmd.Flags().Changed("spot-max-price")

	useSpotInstances := args.useSpotInstances
	spotMaxPrice := args.spotMaxPrice
	if isSpotMaxPriceSet && isSpotSet && !useSpotInstances {
		r.Reporter.Errorf("Can't set max price when not using spot instances")
		os.Exit(1)
	}

	// Validate spot instance are supported
	var isLocalZone bool
	if subnet != "" {
		isLocalZone, err = r.AWSClient.IsLocalAvailabilityZone(availabilityZonesFilter[0])
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}
	if isLocalZone && useSpotInstances {
		r.Reporter.Errorf("Spot instances are not supported for local zones")
		os.Exit(1)
	}

	if !isSpotSet && !isSpotMaxPriceSet && !isLocalZone && interactive.Enabled() {
		useSpotInstances, err = interactive.GetBool(interactive.Input{
			Question: "Use spot instances",
			Help:     cmd.Flags().Lookup("use-spot-instances").Usage,
			Default:  useSpotInstances,
			Required: false,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value for use spot instances: %s", err)
			os.Exit(1)
		}
	}

	if useSpotInstances && !isSpotMaxPriceSet && interactive.Enabled() {
		spotMaxPrice, err = interactive.GetString(interactive.Input{
			Question: "Spot instance max price",
			Help:     cmd.Flags().Lookup("spot-max-price").Usage,
			Required: false,
			Default:  spotMaxPrice,
			Validators: []interactive.Validator{
				spotMaxPriceValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid value for spot max price: %s", err)
			os.Exit(1)
		}
	}

	var maxPrice *float64

	err = spotMaxPriceValidator(spotMaxPrice)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	if spotMaxPrice != "on-demand" {
		price, _ := strconv.ParseFloat(spotMaxPrice, 64)
		maxPrice = &price
	}

	mpBuilder := cmv1.NewMachinePool().
		ID(name).
		InstanceType(instanceType).
		Labels(labelMap).
		Taints(taintBuilders...)

	if autoscaling {
		mpBuilder = mpBuilder.Autoscaling(
			cmv1.NewMachinePoolAutoscaling().
				MinReplicas(minReplicas).
				MaxReplicas(maxReplicas))
	} else {
		mpBuilder = mpBuilder.Replicas(replicas)
	}

	if useSpotInstances {
		spotBuilder := cmv1.NewAWSSpotMarketOptions()
		if maxPrice != nil {
			spotBuilder = spotBuilder.MaxPrice(*maxPrice)
		}
		mpBuilder = mpBuilder.AWS(cmv1.NewAWSMachinePool().
			SpotMarketOptions(spotBuilder))
	}

	// Create a single AZ machine pool for a multi-AZ cluster
	if cluster.MultiAZ() && !multiAZMachinePool && availabilityZone != "" {
		mpBuilder.AvailabilityZones(availabilityZone)
	}

	// Create a single AZ machine pool for a BYOVPC cluster
	if subnet != "" {
		mpBuilder.Subnets(subnet)
	}

	machinePool, err := mpBuilder.Build()
	if err != nil {
		r.Reporter.Errorf("Failed to create machine pool for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	createdMachinePool, err := r.OCMClient.CreateMachinePool(cluster.ID(), machinePool)
	if err != nil {
		r.Reporter.Errorf("Failed to add machine pool to cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if output.HasFlag() {
		if err = output.Print(createdMachinePool); err != nil {
			r.Reporter.Errorf("Unable to print machine pool: %v", err)
			os.Exit(1)
		}
	} else {
		r.Reporter.Infof("Machine pool '%s' created successfully on cluster '%s'", name, clusterKey)
		r.Reporter.Infof("To view all machine pools, run 'rosa list machinepools -c %s'", clusterKey)
	}
}

func Split(r rune) bool {
	return r == '=' || r == ':'
}

// getMachinePoolAvailabilityZones derives the availability zone from the user input or the cluster spec
func getMachinePoolAvailabilityZones(r *rosa.Runtime, cluster *cmv1.Cluster, multiAZMachinePool bool,
	availabilityZoneUserInput string, subnetUserInput string) ([]string, error) {
	// Single AZ machine pool for a multi-AZ cluster
	if cluster.MultiAZ() && !multiAZMachinePool && availabilityZoneUserInput != "" {
		return []string{availabilityZoneUserInput}, nil
	}

	// Single AZ machine pool for a BYOVPC cluster
	if subnetUserInput != "" {
		availabilityZone, err := r.AWSClient.GetSubnetAvailabilityZone(subnetUserInput)
		if err != nil {
			return []string{}, err
		}

		return []string{availabilityZone}, nil
	}

	// Default option of cluster's nodes availability zones
	return cluster.Nodes().AvailabilityZones(), nil
}

func minReplicaValidator(multiAZMachinePool bool) interactive.Validator {
	return func(val interface{}) error {
		minReplicas, err := strconv.Atoi(fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}
		if minReplicas < 0 {
			return fmt.Errorf("min-replicas must be a non-negative integer")
		}
		if multiAZMachinePool && minReplicas%3 != 0 {
			return fmt.Errorf("Multi AZ clusters require that the replicas be a multiple of 3")
		}
		return nil
	}
}

func maxReplicaValidator(minReplicas int, multiAZMachinePool bool) interactive.Validator {
	return func(val interface{}) error {
		maxReplicas, err := strconv.Atoi(fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}
		if minReplicas > maxReplicas {
			return fmt.Errorf("max-replicas must be greater or equal to min-replicas")
		}
		if multiAZMachinePool && maxReplicas%3 != 0 {
			return fmt.Errorf("Multi AZ clusters require that the replicas be a multiple of 3")
		}
		return nil
	}
}

func spotMaxPriceValidator(val interface{}) error {
	spotMaxPrice := fmt.Sprintf("%v", val)
	if spotMaxPrice == "on-demand" {
		return nil
	}
	price, err := strconv.ParseFloat(spotMaxPrice, 64)
	if err != nil {
		return fmt.Errorf("Expected a numeric value for spot max price")
	}

	if price <= 0 {
		return fmt.Errorf("Spot max price must be positive")
	}
	return nil
}

func LabelValidator(val interface{}) error {
	if labels, ok := val.(string); ok {
		_, err := ParseLabels(labels)
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", val)
}

func ParseLabels(labels string) (map[string]string, error) {
	labelMap := make(map[string]string)
	if labels == "" {
		return labelMap, nil
	}
	for _, label := range strings.Split(labels, ",") {
		if !strings.Contains(label, "=") {
			return nil, fmt.Errorf("Expected key=value format for labels")
		}
		tokens := strings.Split(label, "=")
		err := validateLabelKeyValuePair(tokens[0], tokens[1])
		if err != nil {
			return nil, err
		}
		key := strings.TrimSpace(tokens[0])
		value := strings.TrimSpace(tokens[1])
		if _, exists := labelMap[key]; exists {
			return nil, fmt.Errorf("Duplicated label key '%s' used", key)
		}
		labelMap[key] = value
	}
	return labelMap, nil
}

func validateLabelKeyValuePair(key, value string) error {
	if errs := validation.IsQualifiedName(key); len(errs) != 0 {
		return fmt.Errorf("Invalid label key '%s': %s", key, strings.Join(errs, "; "))
	}

	if errs := validation.IsValidLabelValue(value); len(errs) != 0 {
		return fmt.Errorf("Invalid label value '%s': at key: '%s': %s",
			value, key, strings.Join(errs, "; "))
	}
	return nil
}

func taintValidator(val interface{}) error {
	if taints, ok := val.(string); ok {
		_, err := parseTaints(taints)
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", val)
}

func parseTaints(taints string) ([]*cmv1.TaintBuilder, error) {
	taintBuilders := []*cmv1.TaintBuilder{}
	if taints == "" {
		return taintBuilders, nil
	}
	for _, taint := range strings.Split(taints, ",") {
		if !strings.Contains(taint, "=") || !strings.Contains(taint, ":") {
			return nil, fmt.Errorf("Expected key=value:scheduleType format for taints")
		}
		tokens := strings.FieldsFunc(taint, Split)
		taintBuilders = append(taintBuilders, cmv1.NewTaint().Key(tokens[0]).Value(tokens[1]).Effect(tokens[2]))
	}
	return taintBuilders, nil
}

func isBYOVPC(cluster *cmv1.Cluster) bool {
	return len(cluster.AWS().SubnetIDs()) > 0
}
