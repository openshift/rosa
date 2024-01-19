package arguments

import (
	"github.com/spf13/pflag"
)

const (
	deprecatedInstallerRoleArnFlag = "installer-role-arn"
	newInstallerRoleArnFlag        = "role-arn"
)

func NormalizeFlags(f *pflag.FlagSet, name string) pflag.NormalizedName {
	switch name {
	case deprecatedInstallerRoleArnFlag:
		name = newInstallerRoleArnFlag
	}
	return pflag.NormalizedName(name)
}
