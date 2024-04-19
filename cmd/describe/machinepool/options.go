package machinepool

import (
	"fmt"

	"github.com/openshift/rosa/pkg/reporter"
)

type DescribeMachinepoolUserOptions struct {
	machinepool string
}

type DescribeMachinepoolOptions struct {
	reporter *reporter.Object

	args DescribeMachinepoolUserOptions
}

func NewDescribeMachinepoolUserOptions() DescribeMachinepoolUserOptions {
	return DescribeMachinepoolUserOptions{machinepool: ""}
}

func NewDescribeMachinepoolOptions() *DescribeMachinepoolOptions {
	return &DescribeMachinepoolOptions{
		reporter: reporter.CreateReporter(),
		args:     DescribeMachinepoolUserOptions{},
	}
}

func (m *DescribeMachinepoolOptions) Machinepool() string {
	return m.args.machinepool
}

func (m *DescribeMachinepoolOptions) Bind(args DescribeMachinepoolUserOptions) error {
	if args.machinepool == "" {
		return fmt.Errorf("you need to specify a machine pool name")
	}
	m.args.machinepool = args.machinepool
	return nil
}
