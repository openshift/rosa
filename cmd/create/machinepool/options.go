package machinepool

import (
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/reporter"
)

type CreateMachinepoolOptions struct {
	reporter *reporter.Object

	args *mpOpts.CreateMachinepoolUserOptions
}

func NewCreateMachinepoolUserOptions() *mpOpts.CreateMachinepoolUserOptions {
	return &mpOpts.CreateMachinepoolUserOptions{
		InstanceType:          "m5.xlarge",
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

func (m *CreateMachinepoolOptions) Bind(args *mpOpts.CreateMachinepoolUserOptions, argv []string) error {
	m.args = args
	if len(argv) > 0 {
		m.args.Name = argv[0]
	}
	return nil
}
