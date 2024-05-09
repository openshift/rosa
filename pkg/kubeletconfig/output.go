package kubeletconfig

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const emptyName = "-"

func PrintKubeletConfigsForTabularOutput(configs []*cmv1.KubeletConfig) string {
	output := "ID\tNAME\tPOD PIDS LIMIT\n"
	for _, config := range configs {

		name := config.Name()
		if name == "" {
			name = emptyName
		}
		output += fmt.Sprintf("%s\t%s\t%d\n", config.ID(), name, config.PodPidsLimit())
	}

	return output
}
