package kubeletconfig

import (
	"fmt"

	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	promptMessage = "%s the KubeletConfig for cluster '%s' will cause all non-Control Plane " +
		"nodes to reboot. This may cause outages to your applications. Do you wish to continue?"
	abortMessage                     = "%s of KubeletConfig for cluster '%s' aborted."
	OperationDelete KubeletOperation = "delete"
	OperationEdit   KubeletOperation = "edit"
	OperationCreate KubeletOperation = "create"
)

type KubeletOperation string

var (
	singularTense = map[KubeletOperation]string{
		OperationEdit:   "Edit",
		OperationDelete: "Delete",
		OperationCreate: "Create",
	}

	futureTense = map[KubeletOperation]string{
		OperationEdit:   "Editing",
		OperationDelete: "Deleting",
		OperationCreate: "Creating",
	}
)

func PromptUserToAcceptWorkerNodeReboot(operation KubeletOperation, r *rosa.Runtime) bool {
	if !confirm.ConfirmRaw(buildPromptMessage(operation, r.GetClusterKey())) {
		r.Reporter.Infof(buildAbortMessage(operation, r.GetClusterKey()))
		return false
	}
	return true
}

func buildAbortMessage(operation KubeletOperation, clusterKey string) string {
	return fmt.Sprintf(abortMessage, singularTense[operation], clusterKey)
}

func buildPromptMessage(operation KubeletOperation, clusterKey string) string {
	return fmt.Sprintf(promptMessage, futureTense[operation], clusterKey)
}
