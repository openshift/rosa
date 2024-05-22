package addon

import (
	"fmt"

	"github.com/openshift/rosa/pkg/reporter"
)

type DescribeAddonUserOptions struct {
	addon string
}

type DescribeAddonOptions struct {
	reporter *reporter.Object

	args *DescribeAddonUserOptions
}

func NewDescribeAddonUserOptions() *DescribeAddonUserOptions {
	return &DescribeAddonUserOptions{addon: ""}
}

func NewDescribeAddonOptions() *DescribeAddonOptions {
	return &DescribeAddonOptions{
		reporter: reporter.CreateReporter(),
		args:     &DescribeAddonUserOptions{},
	}
}

func (m *DescribeAddonOptions) Addon() string {
	return m.args.addon
}

func (m *DescribeAddonOptions) Bind(args *DescribeAddonUserOptions, argv []string) error {
	m.args = args
	if m.Addon() == "" {
		if len(argv) > 0 {
			m.args.addon = argv[0]
		}
	}
	if args.addon == "" {
		return fmt.Errorf("You need to specify a addon name")
	}
	// if !addon.MachinePoolKeyRE.MatchString(args.addon) {
	// 	return fmt.Errorf("Expected a valid identifier for the addon")
	// }
	m.args.addon = args.addon
	return nil
}
