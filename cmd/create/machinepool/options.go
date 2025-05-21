package machinepool

import (
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/reporter"
)

const instanceType = "m5.xlarge"

type CreateMachinepoolOptions struct {
	reporter reporter.Logger

	args *mpOpts.CreateMachinepoolUserOptions
}

func NewCreateMachinepoolUserOptions() *mpOpts.CreateMachinepoolUserOptions {
	return &mpOpts.CreateMachinepoolUserOptions{
		InstanceType:          instanceType,
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
