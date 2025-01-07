package vpc_client

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	con "github.com/openshift-online/ocm-common/pkg/aws/consts"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (vpc *VPC) DeleteVPCSecurityGroups(customizedOnly bool) error {
	needCleanGroups := []types.SecurityGroup{}
	securityGroups, err := vpc.AWSClient.ListSecurityGroups(vpc.VpcID)
	if customizedOnly {
		for _, sg := range securityGroups {
			for _, tag := range sg.Tags {
				if *tag.Key == "Name" && (*tag.Value == con.ProxySecurityGroupName ||
					*tag.Value == con.AdditionalSecurityGroupName) {
					needCleanGroups = append(needCleanGroups, sg)
				}
			}
		}
	} else {
		needCleanGroups = securityGroups
	}
	if err != nil {
		return err
	}
	for _, sg := range needCleanGroups {
		_, err = vpc.AWSClient.DeleteSecurityGroup(*sg.GroupId)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateAndAuthorizeDefaultSecurityGroupForProxy can prepare a security group for the proxy launch
func (vpc *VPC) CreateAndAuthorizeDefaultSecurityGroupForProxy(ports ...int32) (string, error) {
	var groupID string
	var err error
	var sgIDs []string
	var securityGroupName string
	var securityGroupDescription string
	if len(ports) > 0 {
		securityGroupName = con.BastionSecurityGroupName
		securityGroupDescription = con.BastionSecurityGroupDescription
	} else {
		securityGroupName = con.ProxySecurityGroupName
		securityGroupDescription = con.ProxySecurityGroupDescription
	}
	sgIDs, err = vpc.CreateAdditionalSecurityGroups(1, securityGroupName, securityGroupDescription, ports...)

	if err != nil {
		log.LogError("Security group prepare for proxy failed")
	} else {
		groupID = sgIDs[0]
		log.LogInfo("Authorize SG %s prepared successfully for proxy.", groupID)
	}
	return groupID, err
}

// CreateAdditionalSecurityGroups  can prepare <count> additional security groups
// description can be empty which will be set to default value
// namePrefix is required, otherwise if there is same security group existing the creation will fail
func (vpc *VPC) CreateAdditionalSecurityGroups(count int, namePrefix string, description string, ports ...int32) ([]string, error) {
	preparedSGs := []string{}
	createdsgNum := 0
	if description == "" {
		description = con.DefaultAdditionalSecurityGroupDescription
	}
	for createdsgNum < count {
		sgName := fmt.Sprintf("%s-%d", namePrefix, createdsgNum)
		sg, err := vpc.AWSClient.CreateSecurityGroup(vpc.VpcID, sgName, description)
		if err != nil {
			panic(err)
		}
		groupID := *sg.GroupId
		cidrPortsMap := make(map[string][]int32)
		cidrPortsMap[con.RouteDestinationCidrBlock] = append(cidrPortsMap[con.RouteDestinationCidrBlock], 22)
		if len(ports) > 0 {
			cidrPortsMap[con.RouteDestinationCidrBlock] = append(cidrPortsMap[con.RouteDestinationCidrBlock], ports...)
		} else {
			cidrPortsMap[vpc.CIDRValue] = append(cidrPortsMap[vpc.CIDRValue], 8080)
		}
		for cidr, port := range cidrPortsMap {
			for _, v := range port {
				_, err = vpc.AWSClient.AuthorizeSecurityGroupIngress(groupID, cidr, con.TCPProtocol, v, v)
				if err != nil {
					return preparedSGs, err
				}
			}

		}

		preparedSGs = append(preparedSGs, *sg.GroupId)
		createdsgNum++
	}
	return preparedSGs, nil
}
