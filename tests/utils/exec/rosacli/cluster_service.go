package rosacli

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"k8s.io/apimachinery/pkg/util/wait"

	config "github.com/openshift/rosa/tests/utils/config"
	. "github.com/openshift/rosa/tests/utils/log"
)

type ClusterService interface {
	ResourcesCleaner

	DescribeCluster(clusterID string, flags ...string) (bytes.Buffer, error)
	ReflectClusterDescription(result bytes.Buffer) (*ClusterDescription, error)
	DescribeClusterAndReflect(clusterID string) (*ClusterDescription, error)
	List() (bytes.Buffer, error)
	Create(clusterName string, flags ...string) (bytes.Buffer, error, string)
	DeleteCluster(clusterID string, flags ...string) (bytes.Buffer, error)
	CreateDryRun(clusterName string, flags ...string) (bytes.Buffer, error)
	EditCluster(clusterID string, flags ...string) (bytes.Buffer, error)
	DeleteUpgrade(flags ...string) (bytes.Buffer, error)
	Upgrade(flags ...string) (bytes.Buffer, error)
	DescribeUpgrade(clusterID string, flags ...string) (bytes.Buffer, error)
	DescribeUpgradeAndReflect(clusterID string) (*UpgradeDescription, error)
	InstallLog(clusterID string, flags ...string) (bytes.Buffer, error)
	UnInstallLog(clusterID string, flags ...string) (bytes.Buffer, error)

	IsHostedCPCluster(clusterID string) (bool, error)
	IsSTSCluster(clusterID string) (bool, error)
	IsPrivateCluster(clusterID string) (bool, error)
	IsUsingReusableOIDCConfig(clusterID string) (bool, error)
	GetClusterVersion(clusterID string) (config.Version, error)
	IsBYOVPCCluster(clusterID string) (bool, error)
	IsExternalAuthenticationEnabled(clusterID string) (bool, error)
	GetJSONClusterDescription(clusterID string) (*jsonData, error)
	HibernateCluster(clusterID string, flags ...string) (bytes.Buffer, error)
	ResumeCluster(clusterID string, flags ...string) (bytes.Buffer, error)
	ReflectClusterList(result bytes.Buffer) (clusterList ClusterList, err error)
	WaitClusterStatus(clusterID string, status string, interval int, duration int) error
	WaitClusterDeleted(clusterID string, interval int, duration int) error
}

type clusterService struct {
	ResourcesService
}

func NewClusterService(client *Client) ClusterService {
	return &clusterService{
		ResourcesService: ResourcesService{
			client: client,
		},
	}
}

// Struct for the 'rosa list cluster' output
type ClusterListItem struct {
	ID       string `yaml:"ID,omitempty"`
	Name     string `yaml:"NAME,omitempty"`
	STATE    string `yaml:"STATE,omitempty"`
	TOPOLOGY string `yaml:"TOPOLOGY,omitempty"`
}
type ClusterList struct {
	Clusters []ClusterListItem `yaml:"Clusters,omitempty"`
}

// Struct for the 'rosa describe upgrade' output
type UpgradeDescription struct {
	ID           string `yaml:"ID,omitempty"`
	ClusterID    string `yaml:"Cluster ID,omitempty"`
	NextRun      string `yaml:"Next Run,omitempty"`
	UpgradeState string `yaml:"Upgrade State,omitempty"`
}

// Struct for the 'rosa describe cluster' output
type ClusterDescription struct {
	Name                  string                   `yaml:"Name,omitempty"`
	ID                    string                   `yaml:"ID,omitempty"`
	ExternalID            string                   `yaml:"External ID,omitempty"`
	OpenshiftVersion      string                   `yaml:"OpenShift Version,omitempty"`
	ChannelGroup          string                   `yaml:"Channel Group,omitempty"`
	DNS                   string                   `yaml:"DNS,omitempty"`
	AdditionalPrincipals  string                   `yaml:"Additional Principals,omitempty"`
	AWSAccount            string                   `yaml:"AWS Account,omitempty"`
	AWSBillingAccount     string                   `yaml:"AWS Billing Account,omitempty"`
	APIURL                string                   `yaml:"API URL,omitempty"`
	ConsoleURL            string                   `yaml:"Console URL,omitempty"`
	Region                string                   `yaml:"Region,omitempty"`
	MultiAZ               string                   `yaml:"Multi-AZ,omitempty"`
	State                 string                   `yaml:"State,omitempty"`
	Private               string                   `yaml:"Private,omitempty"`
	Created               string                   `yaml:"Created,omitempty"`
	DetailsPage           string                   `yaml:"Details Page,omitempty"`
	ControlPlane          string                   `yaml:"Control Plane,omitempty"`
	ScheduledUpgrade      string                   `yaml:"Scheduled Upgrade,omitempty"`
	InfraID               string                   `yaml:"Infra ID,omitempty"`
	AdditionalTrustBundle string                   `yaml:"Additional trust bundle,omitempty"`
	Ec2MetadataHttpTokens string                   `yaml:"Ec2 Metadata Http Tokens,omitempty"`
	Availability          []map[string]string      `yaml:"Availability,omitempty"`
	Nodes                 []map[string]interface{} `yaml:"Nodes,omitempty"`
	Network               []map[string]string      `yaml:"Network,omitempty"`
	Proxy                 []map[string]string      `yaml:"Proxy,omitempty"`
	STSRoleArn            string                   `yaml:"Role (STS) ARN,omitempty"`
	// STSExternalID            string                   `yaml:"STS External ID,omitempty"`
	SupportRoleARN           string              `yaml:"Support Role ARN,omitempty"`
	OperatorIAMRoles         []string            `yaml:"Operator IAM Roles,omitempty"`
	InstanceIAMRoles         []map[string]string `yaml:"Instance IAM Roles,omitempty"`
	ManagedPolicies          string              `yaml:"Managed Policies,omitempty"`
	UserWorkloadMonitoring   string              `yaml:"User Workload Monitoring,omitempty"`
	FIPSMod                  string              `yaml:"FIPS mode,omitempty"`
	OIDCEndpointURL          string              `yaml:"OIDC Endpoint URL,omitempty"`
	PrivateHostedZone        []map[string]string `yaml:"Private Hosted Zone,omitempty"`
	AuditLogForwarding       string              `yaml:"Audit Log Forwarding,omitempty"`
	ProvisioningErrorMessage string              `yaml:"Provisioning Error Message,omitempty"`
	ProvisioningErrorCode    string              `yaml:"Provisioning Error Code,omitempty"`
	LimitedSupport           []map[string]string `yaml:"Limited Support,omitempty"`
	AuditLogRoleARN          string              `yaml:"Audit Log Role ARN,omitempty"`
	FailedInflightChecks     string              `yaml:"Failed Inflight Checks,omitempty"`
	ExternalAuthentication   string              `yaml:"External Authentication,omitempty"`
	EnableDeleteProtection   string              `yaml:"Delete Protection,omitempty"`
}

// Pasrse the result of 'rosa list cluster' to the ClusterList struct
func (c *clusterService) ReflectClusterList(result bytes.Buffer) (clusterList ClusterList, err error) {
	clusterList = ClusterList{}
	theMap := c.client.Parser.TableData.Input(result).Parse().Output()
	for _, cItem := range theMap {
		cluster := &ClusterListItem{}
		err = MapStructure(cItem, cluster)
		if err != nil {
			return
		}
		clusterList.Clusters = append(clusterList.Clusters, *cluster)
	}
	return clusterList, err
}

// Check the cluster with the id exists in the ClusterList
func (clusterList ClusterList) IsExist(clusterID string) (existed bool) {
	existed = false
	for _, c := range clusterList.Clusters {
		if c.ID == clusterID {
			existed = true
			break
		}
	}
	return
}

// Get specified cluster by cluster id
func (clusterList ClusterList) Cluster(clusterID string) (cluster ClusterListItem) {
	for _, c := range clusterList.Clusters {
		if c.ID == clusterID {
			return c
		}
	}
	return
}

// Get specified cluster by cluster name
func (clusterList ClusterList) ClusterByName(clusterName string) (cluster ClusterListItem) {
	for _, c := range clusterList.Clusters {
		if c.Name == clusterName {
			return c
		}
	}
	return
}

func (c *clusterService) DescribeCluster(clusterID string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	describe := c.client.Runner.
		Cmd("describe", "cluster").
		CmdFlags(combflags...)
	return describe.Run()
}

func (c *clusterService) DescribeClusterAndReflect(clusterID string) (res *ClusterDescription, err error) {
	output, err := c.DescribeCluster(clusterID)
	if err != nil {
		return nil, err
	}
	return c.ReflectClusterDescription(output)
}

// Pasrse the result of 'rosa describe cluster' to the RosaClusterDescription struct
func (c *clusterService) ReflectClusterDescription(result bytes.Buffer) (res *ClusterDescription, err error) {
	var data []byte
	res = new(ClusterDescription)
	theMap, err := c.client.
		Parser.
		TextData.
		Input(result).
		Parse().
		TransformOutput(func(str string) (newStr string) {
			// Apply transformation to avoid issue with the list of Inflight checks below
			// It will consider
			newStr = strings.Replace(str, "Failed Inflight Checks:", "Failed Inflight Checks: |", 1)
			newStr = strings.ReplaceAll(newStr, "\t", "  ")
			newStr = strings.ReplaceAll(newStr, "not found: Role name", "not found:Role name")
			return
		}).
		YamlToMap()
	if err != nil {
		return
	}
	data, err = yaml.Marshal(&theMap)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(data, res)
	return res, err
}

func (c *clusterService) List() (bytes.Buffer, error) {
	list := c.client.Runner.Cmd("list", "cluster")
	return list.Run()
}

func (c *clusterService) CreateDryRun(clusterName string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterName, "--dry-run"}, flags...)
	createDryRun := c.client.Runner.
		Cmd("create", "cluster").
		CmdFlags(combflags...)
	return createDryRun.Run()
}

func (c *clusterService) Create(clusterName string, flags ...string) (bytes.Buffer, error, string) {
	combflags := append([]string{"-c", clusterName}, flags...)
	createCommand := c.client.Runner.
		Cmd("create", "cluster").
		CmdFlags(combflags...)
	output, err := createCommand.Run()
	return output, err, createCommand.CMDString()
}

func (c *clusterService) DeleteCluster(clusterID string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	deleteCluster := c.client.Runner.
		Cmd("delete", "cluster").
		CmdFlags(combflags...)
	return deleteCluster.Run()
}

func (c *clusterService) EditCluster(clusterID string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	editCluster := c.client.Runner.
		Cmd("edit", "cluster").
		CmdFlags(combflags...)
	return editCluster.Run()
}

func (c *clusterService) DeleteUpgrade(flags ...string) (bytes.Buffer, error) {
	DeleteUpgrade := c.client.Runner.
		Cmd("delete", "upgrade").
		CmdFlags(flags...)
	return DeleteUpgrade.Run()
}

func (c *clusterService) Upgrade(flags ...string) (bytes.Buffer, error) {
	upgrade := c.client.Runner.
		Cmd("upgrade", "cluster").
		CmdFlags(flags...)
	return upgrade.Run()
}

func (c *clusterService) DescribeUpgrade(clusterID string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	describe := c.client.Runner.
		Cmd("describe", "upgrade").
		CmdFlags(combflags...)
	return describe.Run()
}

func (c *clusterService) DescribeUpgradeAndReflect(clusterID string) (res *UpgradeDescription, err error) {
	output, err := c.DescribeUpgrade(clusterID)
	if err != nil {
		return nil, err
	}
	return c.ReflectUpgradeDescription(output)
}

func (c *clusterService) ReflectUpgradeDescription(result bytes.Buffer) (res *UpgradeDescription, err error) {
	var data []byte
	res = new(UpgradeDescription)
	theMap, err := c.client.
		Parser.
		TextData.
		Input(result).
		Parse().
		YamlToMap()
	if err != nil {
		return
	}
	data, err = yaml.Marshal(&theMap)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(data, res)
	return res, err
}

func (c *clusterService) InstallLog(clusterID string, flags ...string) (bytes.Buffer, error) {
	installLog := c.client.Runner.
		Cmd("logs", "install", "-c", clusterID).
		CmdFlags(flags...)
	return installLog.Run()
}
func (c *clusterService) UnInstallLog(clusterID string, flags ...string) (bytes.Buffer, error) {
	UnInstallLog := c.client.Runner.
		Cmd("logs", "uninstall", "-c", clusterID).
		CmdFlags(flags...)
	return UnInstallLog.Run()
}

func (c *clusterService) CleanResources(clusterID string) (errors []error) {
	Logger.Debugf("Nothing releated to cluster was done there")
	return
}

// Check if the cluster is hosted-cp cluster
func (c *clusterService) IsHostedCPCluster(clusterID string) (bool, error) {
	jsonData, err := c.GetJSONClusterDescription(clusterID)
	if err != nil {
		return false, err
	}
	return jsonData.DigBool("hypershift", "enabled"), nil
}

// Check if the cluster is sts cluster. hosted-cp cluster is also treated as sts cluster
func (c *clusterService) IsSTSCluster(clusterID string) (bool, error) {
	jsonData, err := c.GetJSONClusterDescription(clusterID)
	if err != nil {
		return false, err
	}
	return jsonData.DigBool("aws", "sts", "enabled"), nil
}

// Check if the cluster is private cluster
func (c *clusterService) IsPrivateCluster(clusterID string) (bool, error) {
	jsonData, err := c.GetJSONClusterDescription(clusterID)
	if err != nil {
		return false, err
	}
	return jsonData.DigString("api", "listening") == "internal", nil
}

// Check if the cluster is using reusable oidc-config
func (c *clusterService) IsUsingReusableOIDCConfig(clusterID string) (bool, error) {
	jsonData, err := c.GetJSONClusterDescription(clusterID)
	if err != nil {
		return false, err
	}
	return jsonData.DigBool("aws", "sts", "oidc_config", "reusable"), nil
}

// Get cluster version
func (c *clusterService) GetClusterVersion(clusterID string) (clusterVersion config.Version, err error) {
	var clusterConfig *config.ClusterConfig
	clusterConfig, err = config.ParseClusterProfile()
	if err != nil {
		return
	}

	if clusterConfig.Version.RawID != "" {
		clusterVersion = *clusterConfig.Version
	} else {
		// Else retrieve from cluster description
		var jsonData *jsonData
		jsonData, err = c.GetJSONClusterDescription(clusterID)
		if err != nil {
			return
		}
		clusterVersion = config.Version{
			RawID:        jsonData.DigString("version", "raw_id"),
			ChannelGroup: jsonData.DigString("version", "channel_group"),
		}
	}
	return
}

func (c *clusterService) GetJSONClusterDescription(clusterID string) (*jsonData, error) {
	c.client.Runner.JsonFormat()
	output, err := c.DescribeCluster(clusterID)
	if err != nil {
		Logger.Errorf("it met error when describeCluster in IsUsingReusableOIDCConfig is %v", err)
		return nil, err
	}
	c.client.Runner.UnsetFormat()
	return c.client.Parser.JsonData.Input(output).Parse(), nil
}

// Check if the cluster is byo vpc cluster
func (c *clusterService) IsBYOVPCCluster(clusterID string) (bool, error) {
	jsonData, err := c.GetJSONClusterDescription(clusterID)
	if err != nil {
		return false, err
	}
	if len(jsonData.DigString("aws", "subnet_ids")) > 0 {
		return true, nil
	}
	return false, nil
}

func RetrieveDesiredComputeNodes(clusterDescription *ClusterDescription) (nodesNb int, err error) {
	if clusterDescription.Nodes[0]["Compute (desired)"] != nil {
		var isInt bool
		nodesNb, isInt = clusterDescription.Nodes[0]["Compute (desired)"].(int)
		if !isInt {
			err = fmt.Errorf("'%v' is not an integer value", isInt)
		}
	} else {
		// Try autoscale one
		autoscaleInfo := clusterDescription.Nodes[0]["Compute (Autoscaled)"].(string)
		nodesNb, err = strconv.Atoi(strings.Split(autoscaleInfo, "-")[0])
	}
	return
}

// Check if the cluster is external authentication enabled cluster
func (c *clusterService) IsExternalAuthenticationEnabled(clusterID string) (bool, error) {
	jsonData, err := c.GetJSONClusterDescription(clusterID)
	if err != nil {
		return false, err
	}
	return jsonData.DigBool("external_auth_config", "enabled"), nil
}

func (c *clusterService) HibernateCluster(clusterID string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	hibernate := c.client.Runner.
		Cmd("hibernate", "cluster").
		CmdFlags(combflags...)
	return hibernate.Run()
}

func (c *clusterService) ResumeCluster(clusterID string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	resume := c.client.Runner.
		Cmd("resume", "cluster").
		CmdFlags(combflags...)
	return resume.Run()
}

// Wait cluster to some status, the inerval and duration are using minute
func (c *clusterService) WaitClusterStatus(clusterID string, status string, interval int, duration int) error {
	err := wait.PollUntilContextTimeout(
		context.Background(),
		time.Duration(interval)*time.Minute,
		time.Duration(duration)*time.Minute,
		false,
		func(context.Context) (bool, error) {
			clusterListB, err := c.List()
			if err != nil {
				return false, err
			}
			clusterList, err := c.ReflectClusterList(clusterListB)
			if err != nil {
				return false, err
			}
			clusterItem := clusterList.Cluster(clusterID)
			if err != nil {
				return false, err
			}
			if clusterItem.STATE == status {
				return true, nil
			}
			return false, err
		})
	return err
}

// Wait for cluster deleted
func (c *clusterService) WaitClusterDeleted(clusterID string, interval int, duration int) error {
	err := wait.PollUntilContextTimeout(
		context.Background(),
		time.Duration(interval)*time.Minute,
		time.Duration(duration)*time.Minute,
		false,
		func(context.Context) (bool, error) {
			clusterListB, err := c.List()
			if err != nil {
				return false, err
			}
			clusterList, err := c.ReflectClusterList(clusterListB)
			if err != nil {
				return false, err
			}
			return !clusterList.IsExist(clusterID), err
		})
	return err
}
