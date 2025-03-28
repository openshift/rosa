package aws_client

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	CON "github.com/openshift-online/ocm-common/pkg/aws/consts"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (client *AWSClient) CreateRouteTable(vpcID string) (*ec2.CreateRouteTableOutput, error) {
	inputCreateRouteTable := &ec2.CreateRouteTableInput{
		VpcId:             aws.String(vpcID),
		DryRun:            nil,
		TagSpecifications: nil,
	}

	respCreateRT, err := client.Ec2Client.CreateRouteTable(context.TODO(), inputCreateRouteTable)
	if err != nil {
		log.LogError("Create route table failed " + err.Error())
		return nil, err
	}
	err = client.WaitForResourceExisting(*respCreateRT.RouteTable.RouteTableId, 20)
	return respCreateRT, err
}

func (client *AWSClient) AssociateRouteTable(routeTableID string, subnetID string, vpcID string) (*ec2.AssociateRouteTableOutput, error) {
	inputAssociateRouteTable := &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(routeTableID),
		DryRun:       nil,
		GatewayId:    nil,
		SubnetId:     aws.String(subnetID),
	}

	respAssociateRouteTable, err := client.Ec2Client.AssociateRouteTable(context.TODO(), inputAssociateRouteTable)
	if err != nil {
		log.LogError("Associate route table failed " + err.Error())
		return nil, err
	}
	log.LogInfo("Associate route table success " + *respAssociateRouteTable.AssociationId)
	return respAssociateRouteTable, err
}

// ListRouteTable will list all of the route tables created based on the VPC
func (client *AWSClient) ListCustomerRouteTables(vpcID string) ([]types.RouteTable, error) {
	vpcFilterName := "vpc-id"
	Filters := []types.Filter{
		types.Filter{
			Name: &vpcFilterName,
			Values: []string{
				vpcID,
			},
		},
	}
	ListRouteTable := &ec2.DescribeRouteTablesInput{
		Filters: Filters,
	}
	resp, err := client.Ec2Client.DescribeRouteTables(context.TODO(), ListRouteTable)
	if err != nil {
		return nil, err
	}
	customRouteTables := []types.RouteTable{}
	for _, rt := range resp.RouteTables {
		isMain := false
		for _, rta := range rt.Associations {
			if *rta.Main {
				isMain = true
				log.LogInfo("Got main association for rt %s", *rt.RouteTableId)
			}
		}
		if !isMain {
			customRouteTables = append(customRouteTables, rt)
			log.LogInfo("Got custom rt %s ", *rt.RouteTableId)
		}
	}
	return customRouteTables, nil
}

func (client *AWSClient) ListRTAssociations(routeTableID string) ([]string, error) {
	associations := []string{}
	ListRouteTable := &ec2.DescribeRouteTablesInput{
		RouteTableIds: []string{routeTableID},
	}
	resp, err := client.Ec2Client.DescribeRouteTables(context.TODO(), ListRouteTable)
	if err != nil {
		return associations, err
	}
	for _, rt := range resp.RouteTables {
		for _, rta := range rt.Associations {
			associations = append(associations, *rta.RouteTableAssociationId)
		}
	}
	return associations, err
}

func (client *AWSClient) DisassociateRouteTableAssociation(associationID string) (*ec2.DisassociateRouteTableOutput, error) {
	input := &ec2.DisassociateRouteTableInput{
		AssociationId: aws.String(associationID),
		DryRun:        nil,
	}

	resp, err := client.Ec2Client.DisassociateRouteTable(context.TODO(), input)
	if err != nil {
		log.LogError("Disassociate route table failed " + err.Error())
		return nil, err
	}
	log.LogInfo("Disassociate route table success " + associationID)
	return resp, err
}

func (client *AWSClient) DisassociateRouteTableAssociations(routeTableID string) error {
	associationIDs, err := client.ListRTAssociations(routeTableID)
	if err != nil {
		err = fmt.Errorf("List associations of route table %s failed: %s", routeTableID, err)
		return err
	}
	for _, assoID := range associationIDs {
		_, err = client.DisassociateRouteTableAssociation(assoID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (client *AWSClient) CreateRoute(routeTableID string, targetID string) (*types.Route, error) {
	prefix := strings.Split(targetID, "-")[0]
	route := &types.Route{}
	createRouteInput := &ec2.CreateRouteInput{
		RouteTableId:         aws.String(routeTableID),
		DestinationCidrBlock: aws.String(CON.RouteDestinationCidrBlock),
	}
	switch prefix {
	case "cagw":
		createRouteInput.CarrierGatewayId = &targetID
		route.CarrierGatewayId = &targetID
	case "eigw":
		createRouteInput.EgressOnlyInternetGatewayId = &targetID
		route.EgressOnlyInternetGatewayId = &targetID
	case "vpce":
		createRouteInput.LocalGatewayId = &targetID
		route.LocalGatewayId = &targetID
	case "i":
		createRouteInput.InstanceId = &targetID
		route.InstanceId = &targetID
	case "igw":
		createRouteInput.GatewayId = &targetID
		route.GatewayId = &targetID
	case "nat":
		createRouteInput.NatGatewayId = &targetID
		route.NatGatewayId = &targetID
	case "eni":
		createRouteInput.NetworkInterfaceId = &targetID
		route.NetworkInterfaceId = &targetID
	case "tgw":
		createRouteInput.TransitGatewayId = &targetID
		route.TransitGatewayId = &targetID
	default:
		return nil, fmt.Errorf("the type %s is not define in the route creation func, please define it in CreateRoute", prefix)
	}

	_, err := client.Ec2Client.CreateRoute(context.TODO(), createRouteInput)
	if err != nil {
		log.LogError("Create route failed " + err.Error())
		return nil, err
	}
	log.LogInfo("Create route success for route table: " + routeTableID)
	return route, err
}

func (client *AWSClient) DeleteRouteTable(routeTableID string) error {
	input := &ec2.DeleteRouteTableInput{
		RouteTableId: &routeTableID,
	}
	_, err := client.Ec2Client.DeleteRouteTable(context.TODO(), input)
	if err != nil {
		return err
	}
	err = client.WaitForResourceDeleted(routeTableID, 10)
	return err
}
func (client *AWSClient) DeleteRouteTableChain(routeTableID string) error {
	err := client.DisassociateRouteTableAssociations(routeTableID)
	if err != nil {
		return err
	}
	err = client.DeleteRouteTable(routeTableID)
	if err != nil {
		log.LogError("Delete route table %s chain failed %s", routeTableID, err)
	} else {
		log.LogInfo("Delete route table %s chain successfully %s", routeTableID, err)
	}
	return err
}
