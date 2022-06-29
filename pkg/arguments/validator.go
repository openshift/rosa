// This file defines an object to check if Flags are valid.

package arguments

import (
	"fmt"

	"github.com/spf13/pflag"
)

type FlagCheck struct {
	validFlags map[string]struct{}
}

func NewFlagCheck(flags *pflag.FlagSet) *FlagCheck {
	flagCheck := FlagCheck{
		validFlags: map[string]struct{}{},
	}

	flags.VisitAll(func(flag *pflag.Flag) {
		flagCheck.AddValidFlag(flag.Name)
	})
	return &flagCheck
}

func (f *FlagCheck) AddValidFlag(flagName string) *FlagCheck {
	f.validFlags[flagName] = struct{}{}
	return f
}

func (f *FlagCheck) IsValidFlag(flagName string) bool {
	_, found := f.validFlags[flagName]
	return found
}

func (f *FlagCheck) ValidateFlags(flags *pflag.FlagSet) error {
	var invalidFlags string
	flags.VisitAll(func(flag *pflag.Flag) {
		if !f.IsValidFlag(flag.Name) {
			invalidFlags += fmt.Sprintf("%q, ", flag.Name)
		}
	})
	if invalidFlags != "" {
		return fmt.Errorf("Invalid flags: %s", invalidFlags[:len(invalidFlags)-2])
	}
	return nil
}
