// This file defines an object to check if Flags are valid.

package arguments

import "github.com/spf13/pflag"

type FlagCheck struct {
	validFlags map[string]struct{}
}

func NewFlagCheck() *FlagCheck {
	return &FlagCheck{
		validFlags: map[string]struct{}{},
	}
}

func (f *FlagCheck) AddValidFlag(flag *pflag.Flag) *FlagCheck {
	f.validFlags[flag.Name] = struct{}{}
	return f
}

func (f *FlagCheck) AddValidParameter(parameterName string) *FlagCheck {
	f.validFlags[parameterName] = struct{}{}
	return f
}

func (f *FlagCheck) IsValidFlag(flag *pflag.Flag) bool {
	_, found := f.validFlags[flag.Name]
	return found
}

func (f *FlagCheck) IsValidParameter(parameterName string) bool {
	_, found := f.validFlags[parameterName]
	return found
}
