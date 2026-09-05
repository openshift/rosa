package network

import (
	"fmt"
	"os"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/interactive"
	helper "github.com/openshift/rosa/pkg/network"
	opts "github.com/openshift/rosa/pkg/options/network"
	"github.com/openshift/rosa/pkg/rosa"
)

// hfEnabled, hfExitFn, and hfCreateNetwork are package-level
// vars so tests can stub the hyperfleet dispatch path.
var (
	hfEnabled      = hyperfleet.Enabled
	hfExitFn       = func(code int) { os.Exit(code) }
	hfCreateNetwork = func(userOptions *opts.NetworkUserOptions, argv []string) {
		r := rosa.NewRuntime().WithAWS()
		defer r.Cleanup()
		runHyperfleetCreateNetwork(r, userOptions, argv)
	}
)

// runHyperfleetCreateNetwork creates a VPC and private hosted zone for HCP clusters via CloudFormation.
// The hosted zone is named {cluster-name}.hypershift.local and is associated with the VPC.
func runHyperfleetCreateNetwork(r *rosa.Runtime, userOptions *opts.NetworkUserOptions, argv []string) {
	userOptions.CleanTemplateDir()

	// Parse parameters and tags
	parsedParams, parsedTags, err := helper.ParseParams(userOptions.Params)
	if err != nil {
		r.Reporter.Errorf("Failed to parse parameters: %v", err)
		hfExitFn(1)
		return
	}

	// Require ClusterName parameter for hosted zone creation
	clusterName := parsedParams["ClusterName"]
	if clusterName == "" {
		r.Reporter.Errorf("--param ClusterName=<name> is required for hyperfleet network creation")
		r.Reporter.Infof("The cluster name is used to create the private hosted zone: <cluster-name>.hypershift.local")
		hfExitFn(1)
		return
	}

	// Set default stack name if not provided
	if parsedParams["Name"] == "" {
		parsedParams["Name"] = fmt.Sprintf("rosa-hcp-network-%s", clusterName)
		r.Reporter.Infof("Stack name not provided, using default: %s", parsedParams["Name"])
	}

	// Set default region if not provided
	if parsedParams["Region"] == "" {
		parsedParams["Region"] = r.AWSClient.GetRegion()
		r.Reporter.Infof("Region not provided, using: %s", parsedParams["Region"])
	}

	// Add cluster tag to all resources
	if parsedTags == nil {
		parsedTags = make(map[string]string)
	}
	parsedTags[fmt.Sprintf("kubernetes.io/cluster/%s", clusterName)] = "owned"

	// Get mode
	mode, err := interactive.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		hfExitFn(1)
		return
	}

	// Create the CloudFormation template with hosted zone
	templateBodyStr := buildHCPNetworkTemplate(clusterName)
	templateBodyBytes := []byte(templateBodyStr)

	service := helper.NewNetworkService()

	switch mode {
	case interactive.ModeManual:
		r.Reporter.Infof(helper.ManualModeHelperMessage(parsedParams, parsedTags))
		r.Reporter.Infof("\nTo create the network stack manually, save the template below and use:")
		r.Reporter.Infof("  aws cloudformation create-stack --stack-name %s --template-body file://template.yaml --parameters ...", parsedParams["Name"])
		fmt.Println("\nCloudFormation Template:")
		fmt.Println("---")
		fmt.Println(templateBodyStr)
		return

	default:
		r.Reporter.Infof("Creating network stack for HCP cluster '%s'", clusterName)
		r.Reporter.Infof("  Stack Name: %s", parsedParams["Name"])
		r.Reporter.Infof("  Region: %s", parsedParams["Region"])
		r.Reporter.Infof("  Hosted Zone: %s.hypershift.local", clusterName)

		// Create stack using CloudFormation
		templateFile := "" // Empty means use templateBody directly
		err = service.CreateStack(&templateFile, &templateBodyBytes, parsedParams, parsedTags)
		if err != nil {
			r.Reporter.Errorf("Failed to create network stack: %v", err)
			hfExitFn(1)
			return
		}

		r.Reporter.Infof("Network stack created successfully")
		r.Reporter.Infof("Use the following to get stack outputs:")
		r.Reporter.Infof("  aws cloudformation describe-stacks --stack-name %s --query 'Stacks[0].Outputs'", parsedParams["Name"])
	}
}

// buildHCPNetworkTemplate returns a CloudFormation template that creates:
// - VPC with public and private subnets
// - Internet Gateway and NAT Gateway
// - Route53 private hosted zone for the cluster
func buildHCPNetworkTemplate(clusterName string) string {
	return fmt.Sprintf(`
AWSTemplateFormatVersion: '2010-09-09'
Description: CloudFormation template to create VPC and Route53 private hosted zone for ROSA HCP cluster

Parameters:
  Name:
    Type: String
    Description: "Stack name"
  Region:
    Type: String
    Description: "AWS Region"
    Default: "us-east-1"
  VpcCidr:
    Type: String
    Description: "CIDR block for the VPC"
    Default: '10.0.0.0/16'
  ClusterName:
    Type: String
    Description: "Cluster name for hosted zone"
    Default: "%s"

Resources:
  VPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: !Ref VpcCidr
      EnableDnsSupport: true
      EnableDnsHostnames: true
      Tags:
        - Key: Name
          Value: !Ref Name
        - Key: 'kubernetes.io/cluster/%s'
          Value: 'owned'
        - Key: 'service'
          Value: 'ROSA'

  InternetGateway:
    Type: AWS::EC2::InternetGateway
    Properties:
      Tags:
        - Key: Name
          Value: !Sub "${Name}-igw"

  AttachGateway:
    Type: AWS::EC2::VPCGatewayAttachment
    Properties:
      VpcId: !Ref VPC
      InternetGatewayId: !Ref InternetGateway

  PublicSubnet:
    Type: AWS::EC2::Subnet
    Properties:
      VpcId: !Ref VPC
      CidrBlock: !Select [0, !Cidr [!Ref VpcCidr, 4, 8]]
      MapPublicIpOnLaunch: true
      Tags:
        - Key: Name
          Value: !Sub "${Name}-public-subnet"

  PrivateSubnet:
    Type: AWS::EC2::Subnet
    Properties:
      VpcId: !Ref VPC
      CidrBlock: !Select [1, !Cidr [!Ref VpcCidr, 4, 8]]
      MapPublicIpOnLaunch: false
      Tags:
        - Key: Name
          Value: !Sub "${Name}-private-subnet"

  PublicRouteTable:
    Type: AWS::EC2::RouteTable
    Properties:
      VpcId: !Ref VPC
      Tags:
        - Key: Name
          Value: !Sub "${Name}-public-rt"

  PublicRoute:
    Type: AWS::EC2::Route
    DependsOn: AttachGateway
    Properties:
      RouteTableId: !Ref PublicRouteTable
      DestinationCidrBlock: 0.0.0.0/0
      GatewayId: !Ref InternetGateway

  PublicSubnetRouteTableAssociation:
    Type: AWS::EC2::SubnetRouteTableAssociation
    Properties:
      SubnetId: !Ref PublicSubnet
      RouteTableId: !Ref PublicRouteTable

  NatGatewayEIP:
    Type: AWS::EC2::EIP
    DependsOn: AttachGateway
    Properties:
      Domain: vpc

  NatGateway:
    Type: AWS::EC2::NatGateway
    Properties:
      AllocationId: !GetAtt NatGatewayEIP.AllocationId
      SubnetId: !Ref PublicSubnet
      Tags:
        - Key: Name
          Value: !Sub "${Name}-nat"

  PrivateRouteTable:
    Type: AWS::EC2::RouteTable
    Properties:
      VpcId: !Ref VPC
      Tags:
        - Key: Name
          Value: !Sub "${Name}-private-rt"

  PrivateRoute:
    Type: AWS::EC2::Route
    Properties:
      RouteTableId: !Ref PrivateRouteTable
      DestinationCidrBlock: 0.0.0.0/0
      NatGatewayId: !Ref NatGateway

  PrivateSubnetRouteTableAssociation:
    Type: AWS::EC2::SubnetRouteTableAssociation
    Properties:
      SubnetId: !Ref PrivateSubnet
      RouteTableId: !Ref PrivateRouteTable

  S3VPCEndpoint:
    Type: AWS::EC2::VPCEndpoint
    Properties:
      VpcId: !Ref VPC
      ServiceName: !Sub "com.amazonaws.${Region}.s3"
      VpcEndpointType: Gateway
      RouteTableIds:
        - !Ref PublicRouteTable
        - !Ref PrivateRouteTable

  HypershiftLocalZone:
    Type: AWS::Route53::HostedZone
    Properties:
      Name: !Sub "${ClusterName}.hypershift.local"
      VPCs:
        - VPCId: !Ref VPC
          VPCRegion: !Ref Region
      HostedZoneConfig:
        Comment: !Sub "Private hosted zone for ROSA HCP cluster ${ClusterName}"
      HostedZoneTags:
        - Key: Name
          Value: !Sub "${ClusterName}.hypershift.local"
        - Key: 'kubernetes.io/cluster/%s'
          Value: 'owned'
        - Key: 'service'
          Value: 'ROSA'

Outputs:
  VpcId:
    Description: VPC ID
    Value: !Ref VPC
    Export:
      Name: !Sub "${AWS::StackName}-VpcId"

  PublicSubnetId:
    Description: Public Subnet ID
    Value: !Ref PublicSubnet
    Export:
      Name: !Sub "${AWS::StackName}-PublicSubnetId"

  PrivateSubnetId:
    Description: Private Subnet ID
    Value: !Ref PrivateSubnet
    Export:
      Name: !Sub "${AWS::StackName}-PrivateSubnetId"

  HostedZoneId:
    Description: Route53 Private Hosted Zone ID
    Value: !Ref HypershiftLocalZone
    Export:
      Name: !Sub "${AWS::StackName}-HostedZoneId"

  HostedZoneName:
    Description: Route53 Private Hosted Zone Name
    Value: !Sub "${ClusterName}.hypershift.local"
    Export:
      Name: !Sub "${AWS::StackName}-HostedZoneName"
`, clusterName, clusterName, clusterName)
}
