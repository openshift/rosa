package vpc_client

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	CON "github.com/openshift-online/ocm-common/pkg/aws/consts"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (subnet *Subnet) IsPublic() bool {
	if subnet.RTable == nil {
		return false
	}
	for _, route := range subnet.RTable.Routes {
		if strings.HasPrefix(aws.ToString(route.GatewayId), "igw") {
			// There is no direct way in the AWS API to determine if a subnet is public or private.
			// A public subnet is one which has an internet gateway route
			// we look for the gatewayId and make sure it has the prefix of igw to differentiate
			// from the default in-subnet route which is called "local"
			// or other virtual gateway (starting with vgv)
			// or vpc peering connections (starting with pcx).
			return true
		}
	}
	return false
}

func (subnet *Subnet) IsNatgatwatEnabled() bool {
	if subnet.RTable == nil {
		return false
	}
	for _, route := range subnet.RTable.Routes {
		if route.NatGatewayId != nil {
			return true
		}
	}
	return false
}

// PrepareNatGatway will return a NAT gateway if existing no matter which zone set
// zone only work when create public subnet once no NAT gate way existing
// Will implement zone supporting for nat gateway in future. But for now, there is no requirement
func (vpc *VPC) PrepareNatGatway(zone string) (types.NatGateway, error) {
	var gateWay types.NatGateway
	natGatways, err := vpc.AWSClient.ListNatGateWays(vpc.VpcID)
	if err != nil {
		return gateWay, err
	}
	if len(natGatways) != 0 {
		gateWay = natGatways[0]
		log.LogInfo("Found existing nat gateway: %s", *gateWay.NatGatewayId)
		err = vpc.AWSClient.WaitForResourceExisting(*gateWay.NatGatewayId, 10*60)

	} else {
		allocation, err := vpc.AWSClient.AllocateEIPAddress()
		if err != nil {
			return gateWay, fmt.Errorf("error happened when allocate EIP Address for NAT gateway: %s", err)
		}
		publicSubnet, err := vpc.PreparePublicSubnet(zone)
		if err != nil {
			return gateWay, fmt.Errorf("error happened when prepare public subnet for NAT gateway: %s", err)
		}
		natGatway, err := vpc.AWSClient.CreateNatGateway(publicSubnet.ID, *allocation.AllocationId, vpc.VpcID)
		if err != nil {
			return gateWay, fmt.Errorf("error happened when prepare NAT gateway: %s", err)
		}
		gateWay = *natGatway.NatGateway
	}

	return gateWay, err
}
func (vpc *VPC) PreparePublicSubnet(zone string) (*Subnet, error) {
	if vpc.SubnetList != nil {
		for _, subnet := range vpc.SubnetList {
			if !subnet.Private {
				return subnet, nil
			}
		}
	}
	subnets, err := vpc.ListSubnets()
	if err != nil {
		return nil, fmt.Errorf("error happened when list subnet of VPC: %s. %s", vpc.VpcID, err.Error())
	}
	for _, subnet := range subnets {
		if !subnet.Private && subnet.Zone == zone {
			return subnet, nil
		}
	}
	subnet, err := vpc.CreatePublicSubnet(zone)
	if err != nil {
		return nil, fmt.Errorf("error happened when create public subnet of VPC: %s. %s", vpc.VpcID, err.Error())
	}
	return subnet, nil
}

// CreatePrivateSubnet will create a private subnet
// if natEnabled then , it will prepare a public subnet and create a NATgatway to the public subnet
func (vpc *VPC) CreatePrivateSubnet(zone string, natEnabled bool) (*Subnet, error) {
	subNetName := strings.Join([]string{
		vpc.VPCName,
		"private",
		zone,
	}, "-")
	tags := map[string]string{
		"Name":           subNetName,
		CON.PrivateLBTag: CON.LBTagValue,
	}
	respRouteTable, err := vpc.AWSClient.CreateRouteTable(vpc.VpcID)
	if err != nil {
		return nil, err
	}

	subnet, err := vpc.CreateSubnet(zone)
	if err != nil {
		return nil, err
	}

	_, err = vpc.AWSClient.AssociateRouteTable(*respRouteTable.RouteTable.RouteTableId, subnet.ID, vpc.VpcID)
	if err != nil {
		return nil, err
	}

	subnet.RTable = respRouteTable.RouteTable

	if natEnabled {
		natGateway, err := vpc.PrepareNatGatway(zone)
		if err != nil {
			return nil, fmt.Errorf("prepare nat gateway for private cluster failed. %s", err.Error())
		}
		route, err := vpc.AWSClient.CreateRoute(*respRouteTable.RouteTable.RouteTableId, *natGateway.NatGatewayId)
		if err != nil {
			return subnet, fmt.Errorf("error happens when create route NAT gateway route to subnet: %s, %s", subnet.ID, err.Error())
		}
		subnet.RTable.Routes = append(subnet.RTable.Routes, *route)
	}
	_, err = vpc.AWSClient.TagResource(subnet.ID, tags)
	if err != nil {
		return subnet, fmt.Errorf("tag subnet %s failed:%s", subnet.ID, err)
	}
	return subnet, err
}

// CreatePublicSubnet create one public subnet, and related route table, internet gateway, routes. DO NOT include vpc creation.
// Inputs:
//
//	vpcID should be provided, e.g. "vpc-0287d4a924e9f35d9".
//	allocationID is an eip allocate ID, e.g. "eipalloc-0efc1c0ceff5339a2".
//	region is a string of the AWS region. If this value is empty, the default region is "us-east-2".
//	zone is a string. If this value is empty, the default zone is "a".
//	data a VPC struct containing the ids for recording create resources ids and needed while deleting.
//	subnetCidr is a string, e.g. "10.190.1.0/24".
//
// Output:
//
//	If success, a Subnet struct containing the subnetID, private=false, subnetCidr, region, zone and VpcID.
//	Otherwise, nil and an error from the call.
func (vpc *VPC) CreatePublicSubnet(zone string) (*Subnet, error) {
	subNetName := strings.Join([]string{
		vpc.VPCName,
		"public",
		zone,
	}, "-")
	tags := map[string]string{
		"Name":                 subNetName,
		CON.PublicSubNetTagKey: CON.PublicSubNetTagValue,
		CON.PublicLBTag:        CON.LBTagValue,
	}
	subnet, err := vpc.CreateSubnet(zone)
	if err != nil {
		return nil, fmt.Errorf("create subnet meets error:%s", err)
	}

	respRouteTable, err := vpc.AWSClient.CreateRouteTable(vpc.VpcID)
	if err != nil {
		return nil, fmt.Errorf("create RouteTable failed %s", err.Error())
	}
	_, err = vpc.AWSClient.AssociateRouteTable(*respRouteTable.RouteTable.RouteTableId, subnet.ID, vpc.VpcID)
	if err != nil {
		return nil, fmt.Errorf("associate route table failed %s", err.Error())
	}
	subnet.RTable = respRouteTable.RouteTable
	//data.AssociationRouteTableSubnetIDs.AssociationID = append(data.AssociationRouteTableSubnetIDs.AssociationID, *respAssociateRT.AssociationId)
	igwid, err := vpc.PrepareInternetGateway()
	if err != nil {
		return nil, fmt.Errorf("prepare internet gatway failed for vpc: %s", err)
	}

	route, err := vpc.AWSClient.CreateRoute(*respRouteTable.RouteTable.RouteTableId, igwid)
	if err != nil {
		return nil, fmt.Errorf("create route failed for rt %s: %s", *respRouteTable.RouteTable.RouteTableId, err)
	}
	subnet.RTable.Routes = append(subnet.RTable.Routes, *route)
	subnet.Private = false
	_, err = vpc.AWSClient.TagResource(subnet.ID, tags)
	if err != nil {
		return subnet, fmt.Errorf("tag subnet %s failed:%s", subnet.ID, err)
	}
	return subnet, err
}

// CreatePairSubnet create one public subnet one private subnet, and related route table, internet gateway, nat gateway, routes. DO NOT include vpc creation.
// Inputs:
//
//	zone: which zone you prefer to create the subnets
//
// Output:
//
//	If success, a VPC struct containing the ids of the created resources and nil.
//	Otherwise, nil and an error from the call.
func (vpc *VPC) CreatePairSubnet(zone string) (*VPC, []*Subnet, error) {
	publicSubnet, err := vpc.CreatePublicSubnet(zone)
	if err != nil {
		log.LogError("Create public subnet failed" + err.Error())
		return vpc, nil, err
	}
	privateSubnet, err := vpc.CreatePrivateSubnet(zone, true)
	if err != nil {
		log.LogError("Create private subnet failed" + err.Error())
		return vpc, nil, err
	}
	return vpc, []*Subnet{publicSubnet, privateSubnet}, err
}

// PreparePairSubnetByZone will return current pair subents once existing,
// Otherwise it will create a pair.
// If single one missing, it will create another one based on the zone
func (vpc *VPC) PreparePairSubnetByZone(zone string) (map[string]*Subnet, error) {
	log.LogInfo("Going to prepare proper pair of subnets")
	result := map[string]*Subnet{}
	for _, subnet := range vpc.SubnetList {
		log.LogInfo("Subnet %s in zone: %s, region %s", subnet.ID, subnet.Zone, vpc.Region)
		if subnet.Zone == zone {
			if subnet.Private {
				if _, ok := result["private"]; !ok {
					log.LogInfo("Got private subnet %s and set it to the result", subnet.ID)
					if subnet.IsNatgatwatEnabled() {
						result["private"] = subnet
					}
				}
			} else {
				if _, ok := result["public"]; !ok {
					log.LogInfo("Got public subnet %s and set it to the result", subnet.ID)
					result["public"] = subnet
				}
			}
		}
	}

	if _, ok := result["public"]; !ok {
		log.LogInfo("Got no public subnet for current zone %s, going to create one", zone)
		subnet, err := vpc.CreatePublicSubnet(zone)
		if err != nil {
			log.LogError("Prepare public subnet failed for zone %s: %s", zone, err)
			return nil, fmt.Errorf("prepare public subnet failed for zone %s: %s", zone, err)
		}
		result["public"] = subnet
	}
	if _, ok := result["private"]; !ok {
		log.LogInfo("Got no proper private subnet for current zone %s, going to create one", zone)
		subnet, err := vpc.CreatePrivateSubnet(zone, true)
		if err != nil {
			log.LogError("Prepare private subnet failed for zone %s: %s", zone, err)
			return nil, fmt.Errorf("prepare private subnet failed for zone %s: %s", zone, err)
		}
		result["private"] = subnet
	}

	return result, nil
}

// CreateMultiZoneSubnet create private and public subnet in multi zones, and related route table, internet gateway, nat gateway, routes. DO NOT include vpc creation.
// Inputs:
//
//	vpcID should be provided, e.g. "vpc-0287d4a924e9f35d9".
//	allocationID is an eip allocate ID, e.g. "eipalloc-0efc1c0ceff5339a2".
//	region is a string of the AWS region. If this value is empty, the default region is "us-east-2".
//	zone is a slice. Need provide more than 1 zones.
//	data a VPC struct containing the ids for recording create resources ids and needed while deleting.
//	subnetCidr is a slice, 2 times the number of subnetCidrs compared to the number of zones should be provided.
//
// Output:
//
//	If success, a VPC struct containing the ids of the created resources and nil.
//	Otherwise, nil and an error from the call.
func (vpc *VPC) CreateMultiZoneSubnet(zones ...string) error {
	var wg sync.WaitGroup
	var err error
	for index, zone := range zones {
		wg.Add(1)
		go func(targetzone string, sleeping int) {
			defer wg.Done()
			time.Sleep(time.Duration(sleeping) * 2 * time.Second)
			_, _, innererr := vpc.CreatePairSubnet(targetzone)
			if innererr != nil {
				err = innererr
				log.LogError("Create subnets meets error %s", err.Error())
			}
		}(zone, index)
	}
	wg.Wait()
	return err
}

func (vpc *VPC) CreateSubnet(zone string) (*Subnet, error) {
	if zone == "" {
		zone = CON.DefaultAWSZone
	}

	subnetcidr := vpc.CIDRPool.Allocate().CIDR
	respCreateSubnet, err := vpc.AWSClient.CreateSubnet(vpc.VpcID, zone, subnetcidr)
	if err != nil {
		log.LogError("create subnet error " + err.Error())
		return nil, err
	}
	err = vpc.AWSClient.WaitForResourceExisting(*respCreateSubnet.SubnetId, 4)

	if err != nil {

		return nil, err
	}

	log.LogInfo("Created subnet with ID " + *respCreateSubnet.SubnetId)
	subnet := &Subnet{
		ID:      *respCreateSubnet.SubnetId,
		Private: true,
		Zone:    zone,
		Cidr:    subnetcidr,
		Region:  vpc.Region,
		VpcID:   vpc.VpcID,
	}
	vpc.SubnetList = append(vpc.SubnetList, subnet)
	return subnet, err
}

// ListIndicatedSubnetsByVPC will returns the indicated type of subnets like public, private
func (vpc *VPC) ListIndicatedSubnetsByVPC(private bool) ([]string, error) {
	results := []string{}
	subnetDetails, err := vpc.ListSubnets()
	if err != nil {
		return nil, err
	}
	for _, subnet := range subnetDetails {
		if subnet.Private == private {
			results = append(results, subnet.ID)
		}

	}
	return results, nil
}

// ListIndicatedSubnetsByVPC will returns the indicated type of subnets like public, private
func (vpc *VPC) FindIndicatedSubnetsBysubnets(private bool, subnetIDs ...string) ([]string, error) {
	subnets, err := vpc.ListSubnets()
	if err != nil {
		return nil, err
	}
	results := []string{}
	for _, requiredSubnet := range subnetIDs {
		for _, subnet := range subnets {
			if subnet.ID == requiredSubnet && subnet.Private == private {
				results = append(results, subnet.ID)
			}
		}
	}

	return results, nil
}

func (vpc *VPC) DeleteVPCSubnets() error {
	subnets, err := vpc.AWSClient.ListSubnetByVpcID(vpc.VpcID)
	if err != nil {
		return err
	}
	for _, subnet := range subnets {
		_, err = vpc.AWSClient.DeleteSubnet(*subnet.SubnetId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (vpc *VPC) ListSubnets() ([]*Subnet, error) {
	log.LogInfo("Trying to list subnets of the vpc")
	subnets := []*Subnet{}
	awsSubnets, err := vpc.AWSClient.ListSubnetByVpcID(vpc.VpcID)
	if err != nil {
		log.LogError(err.Error())
		return subnets, err
	}
	log.LogInfo("Got %d subnets", len(awsSubnets))
	for _, sub := range awsSubnets {
		subnetName := getTagName(sub.Tags)

		subnet := NewSubnet().
			SetID(*sub.SubnetId).
			SetZone(*sub.AvailabilityZone).
			SetCidr(*sub.CidrBlock).
			SetVpcID(*sub.VpcId).
			SetName(subnetName).
			SetRegion(vpc.Region)

		subnets = append(subnets, subnet)
		log.LogInfo(*sub.SubnetId + "\t" + *sub.CidrBlock + "\t" + *sub.AvailabilityZone + "\t")

	}
	vpc.SubnetList = subnets
	err = vpc.DescribeSubnetRTMappings(vpc.SubnetList...)
	for _, subnet := range vpc.SubnetList {
		subnet.Private = !subnet.IsPublic()
	}
	return subnets, err
}

// UniqueSubnet will return a unique subnet by the subnetID
// It contains more values including the CIDR values
func (vpc *VPC) UniqueSubnet(subnetID string) *Subnet {
	var subnet *Subnet
	for _, sub := range vpc.SubnetList {
		if sub.ID == subnetID {
			subnet = sub
			break
		}
	}
	return subnet
}

// AllSubnetIDs will return all if the subnet IDs of the vpc instance
func (vpc *VPC) AllSubnetIDs() []string {
	subnetIDs := []string{}
	for _, subnet := range vpc.SubnetList {
		subnetIDs = append(subnetIDs, subnet.ID)
	}
	return subnetIDs
}

// AllPublicSubnetIDs will return all of the subnet IDs in vpc instance
func (vpc *VPC) AllPublicSubnetIDs() []string {
	subnetIDs := []string{}
	for _, subnet := range vpc.SubnetList {
		if !subnet.Private {
			subnetIDs = append(subnetIDs, subnet.ID)
		}
	}
	return subnetIDs
}

// AllPublicSubnetIDs will return all of the subnet IDs in vpc instance
func (vpc *VPC) AllPrivateSubnetIDs() []string {
	subnetIDs := []string{}
	for _, subnet := range vpc.SubnetList {
		if subnet.Private {
			subnetIDs = append(subnetIDs, subnet.ID)
		}
	}
	return subnetIDs
}
