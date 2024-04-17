package vpc_client

import (
	"net"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"
)

// ************************* CIDR Pool *************************
type SubnetCIDR struct {
	Reserved bool
	CIDR     string
	IPNet    *net.IPNet
}

type VPCCIDRPool struct {
	CIDR       string
	Prefix     int
	SubNetPool []*SubnetCIDR
}

// ************************** Subnet ****************************
type Subnet struct {
	ID      string
	Private bool
	Zone    string
	Cidr    string
	Region  string
	VpcID   string
	Name    string
	RTable  *types.RouteTable
}

func NewSubnet() *Subnet {
	return &(Subnet{})
}
func (subnet *Subnet) SetID(id string) *Subnet {
	subnet.ID = id
	return subnet
}

func (subnet *Subnet) SetPrivate(private bool) *Subnet {
	subnet.Private = private
	return subnet
}

func (subnet *Subnet) SetZone(zone string) *Subnet {
	subnet.Zone = zone
	return subnet
}

func (subnet *Subnet) SetVpcID(vpcID string) *Subnet {
	subnet.VpcID = vpcID
	return subnet
}

func (subnet *Subnet) SetRegion(region string) *Subnet {
	subnet.Region = region
	return subnet
}

func (subnet *Subnet) SetCidr(cidr string) *Subnet {
	subnet.Cidr = cidr
	return subnet
}

func (subnet *Subnet) SetName(name string) *Subnet {
	subnet.Name = name
	return subnet
}

// **************************** VPC ******************************
type VPC struct {
	AWSClient  *aws_client.AWSClient
	VpcID      string
	VPCName    string
	CIDRValue  string
	CIDRPool   *VPCCIDRPool
	SubnetList []*Subnet
	Region     string
}

func NewVPC() *VPC {
	vpc := &VPC{}
	return vpc
}

func (vpc *VPC) Name(name string) *VPC {
	vpc.VPCName = name
	return vpc
}
func (vpc *VPC) ID(id string) *VPC {
	vpc.VpcID = id
	return vpc
}

func (vpc *VPC) AWSclient(awsClient *aws_client.AWSClient) *VPC {
	vpc.AWSClient = awsClient
	return vpc
}

func (vpc *VPC) CIDR(cidr string) *VPC {
	vpc.CIDRValue = cidr
	return vpc
}

func (vpc *VPC) CIDRpool(cidrPool *VPCCIDRPool) *VPC {
	vpc.CIDRPool = cidrPool
	return vpc
}
func (vpc *VPC) NewCIDRPool() *VPC {
	vpc.CIDRPool = NewCIDRPool(vpc.CIDRValue)
	return vpc
}

func (vpc *VPC) Subnets(subnet ...*Subnet) *VPC {
	vpc.SubnetList = subnet
	return vpc
}
func (vpc *VPC) SetRegion(region string) *VPC {
	vpc.Region = region
	return vpc
}
