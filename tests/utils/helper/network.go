package helper

func TemplateWithoutRegionParam() string {
	templateContent := `AWSTemplateFormatVersion: '2010-09-09'
Description: CloudFormation template to create a ROSA Quickstart default VPC.
  This CloudFormation template may not work with rosa CLI versions later than 1.2.47. 
  Please ensure that you are using the compatible CLI version before deploying this template.

Parameters:
  AvailabilityZoneCount:
    Type: Number
    Description: "Number of Availability Zones to use"
    Default: 1
    MinValue: 1
    MaxValue: 3
  Name:
    Type: String
    Description: "Name prefix for resources"
  VpcCidr:
    Type: String
    Description: CIDR block for the VPC
    Default: '10.0.0.0/16'
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
        - Key: 'rosa_managed_policies'
          Value: 'true'
        - Key: 'rosa_hcp_policies'
          Value: 'true'
        - Key: 'service'
          Value: 'ROSA'
`
	return templateContent
}

func TemplateWithoutNameParam() string {
	templateContent := `AWSTemplateFormatVersion: '2010-09-09'
Description: CloudFormation template to create a ROSA Quickstart default VPC.
  This CloudFormation template may not work with rosa CLI versions later than 1.2.47. 
  Please ensure that you are using the compatible CLI version before deploying this template.

Parameters:
  AvailabilityZoneCount:
    Type: Number
    Description: "Number of Availability Zones to use"
    Default: 1
    MinValue: 1
    MaxValue: 3
  Region:
    Type: String
    Description: "AWS Region"
    Default: "us-west-2"
  VpcCidr:
    Type: String
    Description: CIDR block for the VPC
    Default: '10.0.0.0/16'
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
        - Key: 'rosa_managed_policies'
          Value: 'true'
        - Key: 'rosa_hcp_policies'
          Value: 'true'
        - Key: 'service'
          Value: 'ROSA'
`
	return templateContent
}

func TemplateWithoutCidrValueForVPC() string {
	templateContent := `AWSTemplateFormatVersion: '2010-09-09'
Description: CloudFormation template to create a ROSA Quickstart default VPC.
  This CloudFormation template may not work with rosa CLI versions later than 1.2.47. 
  Please ensure that you are using the compatible CLI version before deploying this template.

Parameters:
  AvailabilityZoneCount:
    Type: Number
    Description: "Number of Availability Zones to use"
    Default: 1
    MinValue: 1
    MaxValue: 3
  Region:
    Type: String
    Description: "AWS Region"
    Default: "us-west-2"
  Name:
    Type: String
    Description: "Name prefix for resources"
  VpcCidr:
    Type: String
    Description: CIDR block for the VPC
    Default: '10.0.0.0/16'
Resources:
  VPC:
    Type: AWS::EC2::VPC
    Properties:
      EnableDnsSupport: true
      EnableDnsHostnames: true
      Tags:
        - Key: Name
          Value: !Ref Name
        - Key: 'rosa_managed_policies'
          Value: 'true'
        - Key: 'rosa_hcp_policies'
          Value: 'true'
        - Key: 'service'
          Value: 'ROSA'
`
	return templateContent
}

func TemplateForSingleVPC() string {
	templateContent := `AWSTemplateFormatVersion: '2010-09-09'
Description: CloudFormation template to create a ROSA Quickstart default VPC.
  This CloudFormation template may not work with rosa CLI versions later than 1.2.47. 
  Please ensure that you are using the compatible CLI version before deploying this template.

Parameters:
  AvailabilityZoneCount:
    Type: Number
    Description: "Number of Availability Zones to use"
    Default: 1
    MinValue: 1
    MaxValue: 3
  Region:
    Type: String
    Description: "AWS Region"
    Default: "us-west-2"
  Name:
    Type: String
    Description: "Name prefix for resources"
  VpcCidr:
    Type: String
    Description: CIDR block for the VPC
    Default: '10.0.0.0/16'
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
        - Key: 'rosa_managed_policies'
          Value: 'true'
        - Key: 'rosa_hcp_policies'
          Value: 'true'
        - Key: 'service'
          Value: 'ROSA'
`
	return templateContent
}
