package kubeletconfig

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const emptyName = "-"

func PrintKubeletConfigsForTabularOutput(configs []*cmv1.KubeletConfig) string {
	output := "ID\tNAME\tPOD PIDS LIMIT\n"
	for _, config := range configs {
		output += fmt.Sprintf("%s\t%s\t%d\n", config.ID(), getName(config), config.PodPidsLimit())
	}

	return output
}

func getName(config *cmv1.KubeletConfig) string {
	if config.Name() == "" {
		return emptyName
	}
	return config.Name()
}

func PrintKubeletConfigForHcp(config *cmv1.KubeletConfig, nodePools []*cmv1.NodePool) string {
	output := PrintKubeletConfigForClassic(config)
	if len(nodePools) != 0 {
		output += "MachinePools Using This KubeletConfig:\n"
		for _, n := range nodePools {
			output += fmt.Sprintf(" - %s\n", n.ID())
		}
	}

	return output
}

func PrintKubeletConfigForClassic(config *cmv1.KubeletConfig) string {
	return fmt.Sprintf("\n"+
		"Name:                                 %s\n"+
		"Pod Pids Limit:                       %d\n",
		getName(config),
		config.PodPidsLimit(),
	)
}
