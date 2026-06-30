package vpc_client

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/openshift-online/ocm-common/pkg/log"
	sshclient "golang.org/x/crypto/ssh"
)

// These are vars (not consts) so that tests can override them to avoid long sleeps.
var (
	sshMaxAttempts   = 5
	sshRetryInterval = 30 * time.Second
	sshDialTimeout   = 30 * time.Second
)

// Exec_CMD runs a command on a remote host over SSH.
// It retries up to sshMaxAttempts times with sshRetryInterval backoff to handle
// transient connection errors that occur while an instance is still initializing.
// Errors from the remote command itself (non-zero exit) are not retried.
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
		Timeout:         sshDialTimeout,
	}

	for attempt := 1; attempt <= sshMaxAttempts; attempt++ {
		result, err = execSSHCommand(config, addr, cmd)
		if err == nil {
			return result, nil
		}
		// A remote command ran but exited non-zero — this is not a transient
		// connection problem, so retrying would just re-run the command.
		if isSSHExitError(err) {
			return "", err
		}
		if attempt < sshMaxAttempts {
			log.LogWarning("SSH attempt %d/%d failed for %s: %s. Retrying in %s...",
				attempt, sshMaxAttempts, addr, err, sshRetryInterval)
			time.Sleep(sshRetryInterval)
		}
	}
	return "", fmt.Errorf("SSH command failed after %d attempts: %w", sshMaxAttempts, err)
}

func execSSHCommand(config *sshclient.ClientConfig, addr string, cmd string) (string, error) {
	client, err := sshclient.Dial("tcp", addr, config)
	if err != nil {
		return "", fmt.Errorf("failed to dial %s: %w", addr, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return "", fmt.Errorf("failed to run command (stderr: %q): %w", stderrStr, err)
		}
		return "", fmt.Errorf("failed to run command: %w", err)
	}
	return stdout.String(), nil
}

// isSSHExitError reports whether err (possibly wrapped) is an SSH remote
// command exit error. These are not transient and should not be retried.
func isSSHExitError(err error) bool {
	var exitErr *sshclient.ExitError
	return errors.As(err, &exitErr)
}

func publicKeyAuthFunc(keyPath string) (sshclient.AuthMethod, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("ssh key file read failed: %w", err)
	}
	signer, err := sshclient.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("ssh key signer failed: %w", err)
	}
	return sshclient.PublicKeys(signer), nil
}
