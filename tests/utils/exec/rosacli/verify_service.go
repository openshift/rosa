package rosacli

import (
	"bytes"
)

type VerifyService interface {
	VerifyOC() (bytes.Buffer, error)
	VerifyPermissions() (bytes.Buffer, error)
	VerifyQuota() (bytes.Buffer, error)
	VerifyRosaClient() (bytes.Buffer, error)
}

type verifyService struct {
	ResourcesService
}

func NewVerifyService(client *Client) VerifyService {
	return &verifyService{
		ResourcesService: ResourcesService{
			client: client,
		},
	}
}

func (vs *verifyService) VerifyOC() (bytes.Buffer, error) {
	verifyCmd := vs.client.Runner.Cmd("verify", "openshift-client")
	return verifyCmd.Run()
}
func (vs *verifyService) VerifyPermissions() (bytes.Buffer, error) {
	verifyCmd := vs.client.Runner.Cmd("verify", "permissions")
	return verifyCmd.Run()
}
func (vs *verifyService) VerifyQuota() (bytes.Buffer, error) {
	verifyCmd := vs.client.Runner.Cmd("verify", "quota")
	return verifyCmd.Run()
}
func (vs *verifyService) VerifyRosaClient() (bytes.Buffer, error) {
	verifyCmd := vs.client.Runner.Cmd("verify", "rosa-client")
	return verifyCmd.Run()
}
