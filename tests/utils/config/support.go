package config

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/utils/exec/occli"
	"github.com/openshift/rosa/tests/utils/log"
)

// DeployCilium The step is provided via here https://hypershift-docs.netlify.app/how-to/aws/other-sdn-providers/#cilium
// Only for HCP cluster now
func DeployCilium(ocClient *occli.Client, podCIDR string, hostPrefix string, outputDir string,
	kubeconfigFile string) error {
	ciliumVersion := "1.14.5"
	yamlFileNames := []string{
		"cluster-network-03-cilium-ciliumconfigs-crd.yaml",
		"cluster-network-06-cilium-00000-cilium-namespace.yaml",
		"cluster-network-06-cilium-00001-cilium-olm-serviceaccount.yaml",
		"cluster-network-06-cilium-00002-cilium-olm-deployment.yaml",
		"cluster-network-06-cilium-00003-cilium-olm-service.yaml",
		"cluster-network-06-cilium-00004-cilium-olm-leader-election-role.yaml",
		"cluster-network-06-cilium-00005-cilium-olm-role.yaml",
		"cluster-network-06-cilium-00006-leader-election-rolebinding.yaml",
		"cluster-network-06-cilium-00007-cilium-olm-rolebinding.yaml",
		"cluster-network-06-cilium-00008-cilium-cilium-olm-clusterrole.yaml",
		"cluster-network-06-cilium-00009-cilium-cilium-clusterrole.yaml",
		"cluster-network-06-cilium-00010-cilium-cilium-olm-clusterrolebinding.yaml",
		"cluster-network-06-cilium-00011-cilium-cilium-clusterrolebinding.yaml",
	}

	url := "https://raw.githubusercontent.com/isovalent/olm-for-cilium/main/manifests"
	for _, n := range yamlFileNames {
		stdout, err := ocClient.Run(
			fmt.Sprintf("oc apply -f %s/cilium.v%s/%s", url, ciliumVersion, n))
		time.Sleep(3 * time.Second)

		if err != nil {
			if strings.Contains(err.Error(), "Warning") {
				stdout, err = ocClient.Run(
					fmt.Sprintf("oc apply -f %s/cilium.v%s/%s", url, ciliumVersion, n))
			}
			if err != nil {
				log.Logger.Errorf("%s:%s", stdout, err.Error())
				return err
			}
		}
	}

	//Set PODCIDR/HOSTPREFIX in gobal var to replace in cilium.yml
	podCIDRReValue := podCIDR[:(len(podCIDR))-3] + "\\" + podCIDR[(len(podCIDR)-3):]

	//Use the right configuration for each network stack: data/resources/cilium.yaml
	var fileName string = path.Join(config.Test.ResourcesDir, "cilium.yaml")

	resultFile := path.Join(outputDir, "cilium.yaml")

	_, _, err := occli.RunCMD(
		fmt.Sprintf("cat %s | sed -e 's/HOSTPREFIX/%v/g' >> %s", fileName, hostPrefix, resultFile))
	if err != nil {
		return err
	}
	_, _, err = occli.RunCMD(
		fmt.Sprintf("cat %s | sed -e 's/PODCIDR/%s/g' >> %s", resultFile, podCIDRReValue, resultFile))
	if err != nil {
		return err
	}

	stdout, err := ocClient.Run(fmt.Sprintf("oc apply -f %s --kubeconfig %s", resultFile, kubeconfigFile))
	time.Sleep(3 * time.Second)
	if err != nil {
		log.Logger.Errorf("%s", stdout)
		return err
	}

	return err
}

func GetKubeconfigDummyFunc() {
	// TODO: create IDP to get kubeconfig
	// Refer to OCM-9183
}
