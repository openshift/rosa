package machinepool

import (
	"fmt"

	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/reporter"
)

type EditMachinepoolUserOptions struct {
	machinepool          string
	labels               string
	replicas             int
	autoscalingEnabled   bool
	minReplicas          int
	maxReplicas          int
	taints               string
	autorepair           bool
	tuningConfigs        string
	kubeletConfigs       string
	nodeDrainGracePeriod string
	maxSurge             string
	maxUnavailable       string
}

type EditMachinepoolOptions struct {
	reporter *reporter.Object

	args *EditMachinepoolUserOptions
}

func NewEditMachinepoolUserOptions() *EditMachinepoolUserOptions {
	return &EditMachinepoolUserOptions{machinepool: "", labels: ""}
}

func NewEditMachinepoolOptions() *EditMachinepoolOptions {
	return &EditMachinepoolOptions{
		reporter: reporter.CreateReporter(),
		args:     &EditMachinepoolUserOptions{},
	}
}

func (m *EditMachinepoolOptions) Machinepool() string {
	return m.args.machinepool
}

func (m *EditMachinepoolOptions) Bind(args *EditMachinepoolUserOptions, argv []string) error {
	m.args = args
	if m.args.machinepool == "" {
		if len(argv) > 0 {
			m.args.machinepool = argv[0]
		}
	}

	if m.args.machinepool == "" {
		return fmt.Errorf("You need to specify a machine pool name")
	}

	if m.args.labels != "" {
		_, err := mpHelpers.ParseLabels(args.labels)
		if err != nil {
			return err
		}
	}

	if m.args.autoscalingEnabled {
		if m.args.minReplicas <= 0 {
			return fmt.Errorf("Min replicas must be greater than zero when autoscaling is enabled")
		}
		if m.args.maxReplicas <= 0 {
			return fmt.Errorf("Max replicas must be greater than zero when autoscaling is enabled")
		}
		if m.args.minReplicas > m.args.maxReplicas {
			return fmt.Errorf("Min replicas must be less than max replicas")
		}
		if m.args.replicas != 0 {
			return fmt.Errorf("Autoscaling enabled on machine pool '%s'. can't set replicas", m.Machinepool())
		}
	}

	return nil
}
