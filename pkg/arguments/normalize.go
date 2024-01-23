package arguments

import (
	"github.com/spf13/pflag"
)

const (
	deprecatedInstallerRoleArnFlag = "installer-role-arn"
	newInstallerRoleArnFlag        = "role-arn"
	DeprecatedDefaultMPLabelsFlag  = "default-mp-labels"
	NewDefaultMPLabelsFlag         = "worker-mp-labels"
)

func NormalizeFlags(f *pflag.FlagSet, name string) pflag.NormalizedName {
	switch name {
	case deprecatedInstallerRoleArnFlag:
		name = newInstallerRoleArnFlag
	case DeprecatedDefaultMPLabelsFlag:
		name = NewDefaultMPLabelsFlag
	}
	return pflag.NormalizedName(name)
}
