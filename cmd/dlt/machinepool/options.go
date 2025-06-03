package machinepool

import (
	"fmt"

	"github.com/openshift/rosa/pkg/machinepool"
	"github.com/openshift/rosa/pkg/reporter"
)

type DeleteMachinepoolUserOptions struct {
	machinepool string
}

type DeleteMachinepoolOptions struct {
	reporter reporter.Logger

	args *DeleteMachinepoolUserOptions
}

func NewDeleteMachinepoolUserOptions() *DeleteMachinepoolUserOptions {
	return &DeleteMachinepoolUserOptions{machinepool: ""}
}

func NewDeleteMachinepoolOptions() *DeleteMachinepoolOptions {
	return &DeleteMachinepoolOptions{
		reporter: reporter.CreateReporter(),
		args:     &DeleteMachinepoolUserOptions{},
	}
}

func (m *DeleteMachinepoolOptions) Machinepool() string {
	return m.args.machinepool
}

func (m *DeleteMachinepoolOptions) Bind(args *DeleteMachinepoolUserOptions, argv []string) error {
	m.args = args
	if m.Machinepool() == "" {
		if len(argv) > 0 {
			m.args.machinepool = argv[0]
		}
	}
	if args.machinepool == "" {
		return fmt.Errorf("You need to specify a machine pool name")
	}
	if !machinepool.MachinePoolKeyRE.MatchString(args.machinepool) {
		return fmt.Errorf("Expected a valid identifier for the machine pool")
	}
	m.args.machinepool = args.machinepool
	return nil
}
