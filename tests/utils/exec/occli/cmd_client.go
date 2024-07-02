package occli

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
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

type RunCMDError struct {
	Stderr string
	Err    error
	CMD    string
}

func (ocClient Client) Run(cmd string, pipeCommands ...string) (stdout string, err error) {

	var stderr string
	fmt.Println(">> Running CMD: ", cmd)
	var pipeCommand string
	for _, command := range pipeCommands {
		pipeCommand += fmt.Sprintf("|%s", command)
	}
	cmd = fmt.Sprintf("%s --kubeconfig %s %s", cmd, ocClient.KubePath, pipeCommand)
	stdout, stderr, err = RunCMD(cmd)
	if err != nil {
		t := &RunCMDError{Stderr: stderr, Err: err, CMD: cmd}
		err = t.Err
	}
	stdout = strings.TrimSuffix(stdout, "\n")
	fmt.Println(">> Got STDOUT: ", stdout)
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
	return
}

func DummyGetKubeConfigFromEnv() {
	// login with oc cmd and then get the kubeconfig from  ~/.kube/config
	// Refer to OCM-9183
}
