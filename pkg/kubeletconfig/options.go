package kubeletconfig

import (
	"fmt"

	"github.com/spf13/cobra"
)

type KubeletConfigOptions struct {
	Name         string
	PodPidsLimit int
}

func NewKubeletConfigOptions() *KubeletConfigOptions {
	return &KubeletConfigOptions{}
}

func (k *KubeletConfigOptions) AddNameFlag(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.SortFlags = false
	flags.StringVar(
		&k.Name,
		NameOption,
		NameOptionDefaultValue,
		NameOptionUsage)
}

func (k *KubeletConfigOptions) AddAllFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.SortFlags = false
	flags.IntVar(
		&k.PodPidsLimit,
		PodPidsLimitOption,
		PodPidsLimitOptionDefaultValue,
		PodPidsLimitOptionUsage)
	k.AddNameFlag(cmd)
}

// BindFromArgs allows the user to use positional args for the name. The --name flag
// will take precedence
func (k *KubeletConfigOptions) BindFromArgs(args []string) {
	if k.Name == "" {
		if len(args) > 0 {
			k.Name = args[0]
		}
	}
}

func (k *KubeletConfigOptions) ValidateForHypershift() error {
	if k.Name == "" {
		return fmt.Errorf("The --name flag is required for Hosted Control Plane clusters.")
	}
	return nil
}
