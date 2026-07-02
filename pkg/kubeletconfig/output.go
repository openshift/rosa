package kubeletconfig

import (
	"fmt"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const emptyName = "-"

func PrintKubeletConfigsForTabularOutput(configs []*cmv1.KubeletConfig) string {
	var output strings.Builder
	output.WriteString("ID\tNAME\tPOD PIDS LIMIT\n")
	for _, config := range configs {
		fmt.Fprintf(&output, "%s\t%s\t%d\n", config.ID(), getName(config), config.PodPidsLimit())
	}

	return output.String()
}

func getName(config *cmv1.KubeletConfig) string {
	if config.Name() == "" {
		return emptyName
	}
	return config.Name()
}

func PrintKubeletConfigForHcp(config *cmv1.KubeletConfig, nodePools []*cmv1.NodePool) string {
	var output strings.Builder
	output.WriteString(PrintKubeletConfigForClassic(config))
	if len(nodePools) != 0 {
		output.WriteString("MachinePools Using This KubeletConfig:\n")
		for _, n := range nodePools {
			fmt.Fprintf(&output, " - %s\n", n.ID())
		}
	}

	return output.String()
}

func PrintKubeletConfigForClassic(config *cmv1.KubeletConfig) string {
	return fmt.Sprintf("\n"+
		"ID:                                   %s\n"+
		"Name:                                 %s\n"+
		"Pod Pids Limit:                       %d\n",
		config.ID(),
		getName(config),
		config.PodPidsLimit(),
	)
}
