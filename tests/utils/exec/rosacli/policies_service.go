package rosacli

import (
	"bytes"
	"strings"

	"github.com/openshift/rosa/tests/utils/log"
)

type PolicyService interface {
	ResourcesCleaner
	AttachPolicy(roleName string, policyArn []string, flags ...string) (bytes.Buffer, error)
	DetachPolicy(roleName string, policyArn []string, flags ...string) (bytes.Buffer, error)
}

type policyService struct {
	ResourcesService
}

func NewPolicyService(client *Client) PolicyService {
	return &policyService{
		ResourcesService: ResourcesService{
			client: client,
		},
	}
}

// Attach policies to a role
func (ps *policyService) AttachPolicy(roleName string, policyArn []string, flags ...string) (bytes.Buffer, error) {
	attachPolicy := ps.client.Runner.Cmd("attach", "policy")
	// join policyArn with ,
	attachPolicy.CmdFlags(append(flags, "--role-name", roleName, "--policy-arns", strings.Join(policyArn, ","))...)
	return attachPolicy.Run()
}

// Detach policies to a role
func (ps *policyService) DetachPolicy(roleName string, policyArn []string, flags ...string) (bytes.Buffer, error) {
	detachPolicy := ps.client.Runner.Cmd("detach", "policy")
	// join policyArn with ,
	detachPolicy.CmdFlags(append(flags, "--role-name", roleName, "--policy-arns", strings.Join(policyArn, ","))...)
	return detachPolicy.Run()
}

func (ps *policyService) CleanResources(clusterID string) (errors []error) {
	log.Logger.Debugf("Nothing to clean in Version Service")
	return
}
