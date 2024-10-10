package occli

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/utils/helper"
	"github.com/openshift/rosa/tests/utils/log"
)

type Client struct {
	KubePath string
}

func NewOCClient(kubePath ...string) *Client {
	ocClient := &Client{}
	if len(kubePath) > 0 {
		ocClient = &Client{
			KubePath: kubePath[0],
		}
	} else {
		usersValue, err := helper.ReadFileContent(config.Test.ClusterKubeconfigIdpFile)
		if err == nil {
			kubeconfig, err := GetKubeConfigFromEnv(usersValue)
			if err != nil {
				log.Logger.Errorf("Error to get kubeconfig from environment: %s", err)
			} else {
				helper.CreateFileWithContent(config.Test.ClusterKubeconfigFile, kubeconfig)
				ocClient = &Client{
					KubePath: config.Test.ClusterKubeconfigFile,
				}
			}
		}
	}
	return ocClient
}

func (ocClient Client) Run(cmd string, retryTimes int, pipeCommands ...string) (stdout string, err error) {
	var stderr string
	for retryTimes > 0 {
		log.Logger.Info("Running CMD: ", cmd)
		var pipeCommand string
		for _, command := range pipeCommands {
			pipeCommand += fmt.Sprintf("|%s", command)
		}
		cmd = fmt.Sprintf("%s --kubeconfig %s %s", cmd, ocClient.KubePath, pipeCommand)
		stdout, stderr, err = RunCMD(cmd)
		if err != nil {
			log.Logger.Errorf("Got output: %s", stderr)
		} else {
			stdout = strings.TrimSuffix(stdout, "\n")
			log.Logger.Infof("Got output: %s", stdout)
			return
		}
		time.Sleep(3 * time.Second)
		retryTimes--
	}
	return
}

func RunCMD(cmd string) (stdout string, stderr string, err error) {
	var stdoutput bytes.Buffer
	var stderroutput bytes.Buffer
	CMD := exec.Command("bash", "-c", cmd)
	CMD.Stderr = &stderroutput
	CMD.Stdout = &stdoutput
	err = CMD.Run()
	stdout = strings.TrimPrefix(stdoutput.String(), "\n")
	stderr = strings.TrimPrefix(stderroutput.String(), "\n")
	if err != nil {
		err = fmt.Errorf("%s:%s", err.Error(), stderr)
	}
	return
}

func GetKubeConfigFromEnv(usersValue string) (kubeconfig string, err error) {
	// login with oc cmd and then get the kubeconfig from  ~/.kube/config
	// Refer to OCM-9183
	values := strings.Split(usersValue, ":")
	name := values[0]
	password := values[1]

	apiUrl, err := helper.ReadFileContent(config.Test.APIURLFile)
	if err != nil {
		log.Logger.Errorf("%s", err)
		return "", err
	}

	_, stderr, err := RunCMD("oc login " + apiUrl + " -u " + name + " -p " + password)
	if err != nil {
		log.Logger.Errorf("%s, %s", err, stderr)
		return "", err
	}

	stdout, stderr, err := RunCMD("cat ~/.kube/config")
	if err != nil {
		log.Logger.Errorf("%s, %s", err, stderr)
		return "", err
	}

	return stdout, err
}
