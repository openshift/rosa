package machinepool

import (
	"fmt"

	"github.com/openshift/rosa/pkg/reporter"
)

type DescribeMachinepoolUserOptions struct {
	machinepool string
}

type DescribeMachinepoolOptions struct {
	reporter reporter.Logger

	args *DescribeMachinepoolUserOptions
}

func NewDescribeMachinepoolUserOptions() *DescribeMachinepoolUserOptions {
	return &DescribeMachinepoolUserOptions{machinepool: ""}
}

func NewDescribeMachinepoolOptions() *DescribeMachinepoolOptions {
	return &DescribeMachinepoolOptions{
		reporter: reporter.CreateReporter(),
		args:     &DescribeMachinepoolUserOptions{},
	}
}

func (m *DescribeMachinepoolOptions) Machinepool() string {
	return m.args.machinepool
}

func (m *DescribeMachinepoolOptions) Bind(args *DescribeMachinepoolUserOptions, argv []string) error {
	m.args = args
	if m.args.machinepool == "" {
		if len(argv) > 0 {
			m.args.machinepool = argv[0]
		}
	}

	if m.args.machinepool == "" {
		return fmt.Errorf("you need to specify a machine pool name")
	}

	return nil
}
