package vpc_client

import (
	"bytes"
	"fmt"
	"os"

	sshclient "golang.org/x/crypto/ssh"
)

func Exec_CMD(userName, keyPath string, addr string, cmd string) (result string, err error) {
	authMethod, err := publicKeyAuthFunc(keyPath)
	if err != nil {
		return "", err
	}
	config := &sshclient.ClientConfig{
		User: userName,
		Auth: []sshclient.AuthMethod{
			authMethod,
		},
		HostKeyCallback: sshclient.InsecureIgnoreHostKey(),
	}

	client, err := sshclient.Dial("tcp", addr, config)
	if err != nil {
		err = fmt.Errorf("failed to dail %s", err)
		return "", err
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		err = fmt.Errorf("failed to create session %s", err)
		return "", err
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b

	if err := session.Run(cmd); err != nil {
		err = fmt.Errorf("failed to run command %s", err)
		return "", err
	}
	return b.String(), nil
}

func publicKeyAuthFunc(keyPath string) (sshclient.AuthMethod, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		err = fmt.Errorf("ssh key file read failed %s", err)
		return nil, err
	}
	signer, err := sshclient.ParsePrivateKey(key)
	if err != nil {
		err = fmt.Errorf("ssh key sinher failed %s", err)
		return nil, err
	}
	return sshclient.PublicKeys(signer), nil
}
