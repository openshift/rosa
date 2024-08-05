package occli

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

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
		DummyGetKubeConfigFromEnv()
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

func DummyGetKubeConfigFromEnv() {
	// login with oc cmd and then get the kubeconfig from  ~/.kube/config
	// Refer to OCM-9183
}
