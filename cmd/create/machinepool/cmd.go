/*
Copyright (c) 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package machinepool

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
)

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var machinePoolKeyRE = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

var args struct {
	name                  string
	instanceType          string
	replicas              int
	autoscalingEnabled    bool
	minReplicas           int
	maxReplicas           int
	labels                string
	taints                string
	useSpotInstances      bool
	spotMaxPrice          string
	multiAvailabilityZone bool
	availabilityZone      string
}

var Cmd = &cobra.Command{
	Use:     "machinepool",
	Aliases: []string{"machinepools", "machine-pool", "machine-pools"},
	Short:   "Add machine pool to cluster",
	Long:    "Add a machine pool to the cluster.",
	Example: `  # Interactively add a machine pool to a cluster named "mycluster"
  rosa create machinepool --cluster=mycluster --interactive

  # Add a machine pool mp-1 with 3 replicas of m5.xlarge to a cluster
  rosa create machinepool --cluster=mycluster --name=mp-1 --replicas=3 --instance-type=m5.xlarge

  # Add a machine pool mp-1 with autoscaling enabled and 3 to 6 replicas of m5.xlarge to a cluster
  rosa create machinepool --cluster=mycluster --name=mp-1 --enable-autoscaling \
	--min-replicas=3 --max-replicas=6 --instance-type=m5.xlarge

  # Add a machine pool with labels to a cluster
  rosa create machinepool -c mycluster --name=mp-1 --replicas=2 --instance-type=r5.2xlarge --labels=foo=bar,bar=baz,

  # Add a machine pool with spot instances to a cluster
  rosa create machinepool -c mycluster --name=mp-1 --replicas=2 --instance-type=r5.2xlarge --use-spot-instances \
    --spot-max-price=0.5`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)

	flags.StringVar(
		&args.name,
		"name",
		"",
		"Name for the machine pool (required).",
	)

	flags.IntVar(
		&args.replicas,
		"replicas",
		0,
		"Count of machines for the machine pool (required when autoscaling is disabled).",
	)

	flags.BoolVar(
		&args.autoscalingEnabled,
		"enable-autoscaling",
		false,
		"Enable autoscaling for the machine pool.",
	)

	flags.IntVar(
		&args.minReplicas,
		"min-replicas",
		0,
		"Minimum number of machines for the machine pool.",
	)

	flags.IntVar(
		&args.maxReplicas,
		"max-replicas",
		0,
		"Maximum number of machines for the machine pool.",
	)

	flags.StringVar(
		&args.instanceType,
		"instance-type",
		"m5.xlarge",
		"Instance type that should be used.",
	)

	flags.StringVar(
		&args.labels,
		"labels",
		"",
		"Labels for machine pool. Format should be a comma-separated list of 'key=value'. "+
			"This list will overwrite any modifications made to Node labels on an ongoing basis.",
	)

	flags.StringVar(
		&args.taints,
		"taints",
		"",
		"Taints for machine pool. Format should be a comma-separated list of 'key=value:ScheduleType'. "+
			"This list will overwrite any modifications made to Node taints on an ongoing basis.",
	)

	flags.BoolVar(
		&args.useSpotInstances,
		"use-spot-instances",
		false,
		"Use spot instances for the machine pool.",
	)

	flags.StringVar(
		&args.spotMaxPrice,
		"spot-max-price",
		"on-demand",
		"Max price for spot instance. If empty use the on-demand price.",
	)

	flags.BoolVar(
		&args.multiAvailabilityZone,
		"multi-availability-zone",
		true,
		"Create a multi-AZ machine pool for a multi-AZ cluster")

	flags.StringVar(
		&args.availabilityZone,
		"availability-zone",
		"",
		"Select availability zone to create a single AZ machine pool for a multi-AZ cluster")

	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()
	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

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

	if isAvailabilityZoneSet && isMultiAvailabilityZoneSet && args.multiAvailabilityZone {
		r.Reporter.Errorf("Setting the `availability-zone` flag is only supported for creating a single AZ " +
			"machine pool in a multi-AZ cluster")
		os.Exit(1)
	}

	var err error
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

	// Single AZ machine pool for a multi-AZ cluster
	var multiAZMachinePool bool
	var availabilityZone string
	if cluster.MultiAZ() {
		// Choosing a single AZ machine pool implicitly
		if isAvailabilityZoneSet {
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
	// Machine pool instance type:
	instanceType := args.instanceType
	instanceTypeList, err := r.OCMClient.GetAvailableMachineTypes()
	if err != nil {
		r.Reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
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

	labels := args.labels
	if interactive.Enabled() {
		labels, err = interactive.GetString(interactive.Input{
			Question: "Labels",
			Help:     cmd.Flags().Lookup("labels").Usage,
			Default:  labels,
			Validators: []interactive.Validator{
				labelValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
			os.Exit(1)
		}
	}
	labelMap, err := parseLabels(labels)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	taints := args.taints
	if interactive.Enabled() {
		taints, err = interactive.GetString(interactive.Input{
			Question: "Taints",
			Help:     cmd.Flags().Lookup("taints").Usage,
			Default:  taints,
			Validators: []interactive.Validator{
				taintValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
			os.Exit(1)
		}
	}
	taintBuilders, err := parseTaints(taints)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Spot instances
	isSpotSet := cmd.Flags().Changed("use-spot-instances")
	isSpotMaxPriceSet := cmd.Flags().Changed("spot-max-price")

	useSpotInstances := args.useSpotInstances
	spotMaxPrice := args.spotMaxPrice
	if isSpotMaxPriceSet && isSpotSet && !useSpotInstances {
		r.Reporter.Errorf("Can't set max price when not using spot instances")
		os.Exit(1)
	}

	if !isSpotSet && !isSpotMaxPriceSet && interactive.Enabled() {
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
	if cluster.MultiAZ() && !multiAZMachinePool {
		mpBuilder.AvailabilityZones(availabilityZone)
	}

	machinePool, err := mpBuilder.Build()
	if err != nil {
		r.Reporter.Errorf("Failed to create machine pool for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	_, err = r.OCMClient.CreateMachinePool(cluster.ID(), machinePool)
	if err != nil {
		r.Reporter.Errorf("Failed to add machine pool to cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	r.Reporter.Infof("Machine pool '%s' created successfully on cluster '%s'", name, clusterKey)
	r.Reporter.Infof("To view all machine pools, run 'rosa list machinepools -c %s'", clusterKey)
}

func Split(r rune) bool {
	return r == '=' || r == ':'
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

func labelValidator(val interface{}) error {
	if labels, ok := val.(string); ok {
		_, err := parseLabels(labels)
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", val)
}

func parseLabels(labels string) (map[string]string, error) {
	labelMap := make(map[string]string)
	if labels == "" {
		return labelMap, nil
	}
	for _, label := range strings.Split(labels, ",") {
		if !strings.Contains(label, "=") {
			return nil, fmt.Errorf("Expected key=value format for labels")
		}
		tokens := strings.Split(label, "=")
		labelMap[strings.TrimSpace(tokens[0])] = strings.TrimSpace(tokens[1])
	}
	return labelMap, nil
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
