package rosacli

import (
	"bytes"
	"errors"
)

type ResourcesCleaner interface {
	CleanResources(clusterID string) []error
}

type CLDNamedResourceService interface {
	ResourcesCleaner

	List(clusterID string) (bytes.Buffer, error)
	Describe(clusterID string, name string) (bytes.Buffer, error)
	Create(clusterID string, name string, flags ...string) (bytes.Buffer, error)
	Edit(clusterID string, name string, flags ...string) (bytes.Buffer, error)
	Delete(clusterID string, name string) (bytes.Buffer, error)
}

type ResourcesService struct {
	client *Client
}

type Client struct {
	// Clients
	Runner *runner
	Parser *Parser

	// services
	// Keep in alphabetical order
	AutoScaler           AutoScalerService
	BreakGlassCredential BreakGlassCredentialService
	Cluster              ClusterService
	ExternalAuthProvider ExternalAuthProviderService
	IAMServiceAccount    IAMServiceAccountService
	ImageMirror          ImageMirrorService
	IDP                  IDPService
	Ingress              IngressService
	KubeletConfig        KubeletConfigService
	MachinePool          MachinePoolService
	MachinePoolUpgrade   MachinePoolUpgradeService
	NetworkVerifier      NetworkVerifierService
	NetworkResources     NetworkResourcesService
	OCMResource          OCMResourceService
	Policy               PolicyService
	TuningConfig         TuningConfigService
	User                 UserService
	Upgrade              UpgradeService
	Verify               VerifyService
	Version              VersionService
}

func NewClient() *Client {
	runner := NewRunner()
	parser := NewParser()

	client := &Client{
		Runner: runner,
		Parser: parser,
	}

	// Keep in alphabetical order
	client.AutoScaler = NewAutoScalerService(client)
	client.BreakGlassCredential = NewBreakGlassCredentialService(client)
	client.Cluster = NewClusterService(client)
	client.ExternalAuthProvider = NewExternalAuthProviderService(client)
	client.IAMServiceAccount = NewIAMServiceAccountService(client)
	client.ImageMirror = NewImageMirrorService(client)
	client.IDP = NewIDPService(client)
	client.Ingress = NewIngressService(client)
	client.KubeletConfig = NewKubeletConfigService(client)
	client.MachinePool = NewMachinePoolService(client)
	client.MachinePoolUpgrade = NewMachinePoolUpgradeService(client)
	client.NetworkVerifier = NewNetworkVerifierService(client)
	client.NetworkResources = NewNetworkResourceService(client)
	client.OCMResource = NewOCMResourceService(client)
	client.Policy = NewPolicyService(client)
	client.TuningConfig = NewTuningConfigService(client)
	client.User = NewUserService(client)
	client.Upgrade = NewUpgradeService(client)
	client.Verify = NewVerifyService(client)
	client.Version = NewVersionService(client)

	return client
}

func (c *Client) CleanResources(clusterID string) error {
	var errorList []error

	// Keep in logical order
	errorList = append(errorList, c.Version.CleanResources(clusterID)...)
	errorList = append(errorList, c.TuningConfig.CleanResources(clusterID)...)
	errorList = append(errorList, c.MachinePoolUpgrade.CleanResources(clusterID)...)
	errorList = append(errorList, c.MachinePool.CleanResources(clusterID)...)
	errorList = append(errorList, c.Ingress.CleanResources(clusterID)...)
	errorList = append(errorList, c.NetworkVerifier.CleanResources(clusterID)...)
	errorList = append(errorList, c.NetworkResources.CleanResources(clusterID)...)
	errorList = append(errorList, c.KubeletConfig.CleanResources(clusterID)...)
	errorList = append(errorList, c.User.CleanResources(clusterID)...)
	errorList = append(errorList, c.IDP.CleanResources(clusterID)...)
	errorList = append(errorList, c.OCMResource.CleanResources(clusterID)...)
	errorList = append(errorList, c.Cluster.CleanResources(clusterID)...)
	errorList = append(errorList, c.BreakGlassCredential.CleanResources(clusterID)...)
	errorList = append(errorList, c.ExternalAuthProvider.CleanResources(clusterID)...)
	errorList = append(errorList, c.AutoScaler.CleanResources(clusterID)...)
	errorList = append(errorList, c.Policy.CleanResources(clusterID)...)
	errorList = append(errorList, c.Upgrade.CleanResources(clusterID)...)

	return errors.Join(errorList...)

}
