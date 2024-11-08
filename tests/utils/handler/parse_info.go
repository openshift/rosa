package handler

import (
	"encoding/json"
	"os"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/utils/helper"
	"github.com/openshift/rosa/tests/utils/log"
)

// ParseClusterDetail Get the cluster info from cluster-detail.json file
func ParseClusterDetail() (*ClusterDetail, error) {
	var cd *ClusterDetail

	_, err := os.Stat(config.Test.ClusterDetailFile)
	if err != nil {
		log.Logger.Warn("Cluster detail file not exists")
		return nil, nil
	}
	cdContent, err := helper.ReadFileContent(config.Test.ClusterDetailFile)
	if err != nil {
		log.Logger.Errorf("Error happened when read cluster detail: %s", err.Error())
		return nil, err
	}
	err = json.Unmarshal([]byte(cdContent), &cd)
	if err != nil {
		log.Logger.Errorf("Error happened when parse cluster detail file to ClusterDetail struct: %s", err.Error())
		return nil, err
	}
	return cd, err
}
