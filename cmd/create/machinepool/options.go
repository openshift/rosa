package machinepool

import (
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/reporter"
)

type CreateMachinepoolOptions struct {
	reporter reporter.Logger

	args *mpOpts.CreateMachinepoolUserOptions
}

func NewCreateMachinepoolUserOptions() *mpOpts.CreateMachinepoolUserOptions {
	return &mpOpts.CreateMachinepoolUserOptions{
		InstanceType:          mpOpts.DefaultInstanceType,
		AutoscalingEnabled:    false,
		MultiAvailabilityZone: true,
		Autorepair:            true,
	}
}

func NewCreateMachinepoolOptions() *CreateMachinepoolOptions {
	return &CreateMachinepoolOptions{
		reporter: reporter.CreateReporter(),
		args:     &mpOpts.CreateMachinepoolUserOptions{},
	}
}

func (m *CreateMachinepoolOptions) Machinepool() *mpOpts.CreateMachinepoolUserOptions {
	return m.args
}
