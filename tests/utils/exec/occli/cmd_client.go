package occli

import (
	"bytes"
	"fmt"
	"os"
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

func NewOCClient(kubePath ...string) (ocClient *Client, err error) {
	if len(kubePath) > 0 {
		ocClient = &Client{
			KubePath: kubePath[0],
		}
	} else {
		var userValues string
		userValues, err = helper.ReadFileContent(config.Test.ClusterIDPAdminUsernamePassword)
		if err != nil {
			return
		}

		values := strings.Split(userValues, ":")
		user := values[0]
		password := values[1]

		var kubeconfigFile string
		kubeconfigFile, err = LoginToKubeconfigFile(user, password)
		if err != nil {
			log.Logger.Errorf("Failed to get kubeconfig from environment: %s", err)
			return
		} else {
			ocClient = &Client{
				KubePath: kubeconfigFile,
			}
		}
	}
	return ocClient, nil
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

func LoginToKubeconfigFile(user string, password string) (kubeconfigFile string, err error) {
	// Refer to OCM-9183
	retryTimes := 10
	apiUrl, err := helper.ReadFileContent(config.Test.APIURLFile)
	if err != nil {
		log.Logger.Errorf("%s", err)
		return
	}

	f, err := os.CreateTemp(config.Test.ArtifactDir, "temp-")
	if err != nil {
		log.Logger.Errorf("%s", err)
		return
	}
	kubeconfigFile = f.Name()
	f.Close()

	for retryTimes > 0 {
		var stderr string
		_, stderr, err = RunCMD("oc login " + apiUrl + " -u " + user + " -p " + password + " --kubeconfig " + kubeconfigFile)
		if err != nil {
			log.Logger.Errorf("%s, %s", err, stderr)
		} else {
			break
		}
		time.Sleep(10 * time.Second)
		retryTimes--
	}

	return
}
