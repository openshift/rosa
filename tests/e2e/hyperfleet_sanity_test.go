package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	ec2svc "github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	elbsvc "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	elbv2svc "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	iamsvc "github.com/aws/aws-sdk-go-v2/service/iam"
	route53svc "github.com/aws/aws-sdk-go-v2/service/route53"
	route53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	stssvc "github.com/aws/aws-sdk-go-v2/service/sts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	hfrest "github.com/openshift-online/rosa-hyperfleet-api/clientset/rest"

	rosaaws "github.com/openshift/rosa/pkg/aws"
	urlHelper "github.com/openshift/rosa/pkg/helper/url"
	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/tests/ci/labels"
	rosacli "github.com/openshift/rosa/tests/utils/exec/rosacli"
)

const (
	hfVPCReadyTimeout       = 2 * time.Minute
	hfSubnetReadyTimeout    = 2 * time.Minute
	hfClusterReadyInterval  = 30 * time.Second
	hfClusterReadyTimeout   = 90 * time.Minute
	hfNodePoolReadyInterval = 30 * time.Second
	hfNodePoolReadyTimeout  = 30 * time.Minute
	hfDefaultInstanceType   = "m5.xlarge"
	hfMaxClusterNameLength  = 18 // Platform API namespace limit; see hyperfleet-guidelines.md

	// teardownGracePeriod bounds how long each cleanup node may run after a
	// mid-run interrupt (Ctrl-C). Ginkgo's default is 30s, which is far too
	// short for AWS teardown and leaves resources orphaned. It is sized to
	// exceed the longest single cleanup node (cluster delete waits up to
	// hfClusterReadyTimeout, followed by the LB/ENI/instance waits) so the
	// per-resource timeouts govern rather than this cap. It has no effect on
	// a normal run to completion.
	teardownGracePeriod = hfClusterReadyTimeout + 30*time.Minute
)

var _ = Describe("Hyperfleet sanity",
	labels.Hyperfleet.Sanity,
	func() {
		It("creates and deletes an HCP cluster via the Platform API", func(ctx SpecContext) {
			hfURL := os.Getenv("HYPERFLEET_URL")
			if hfURL == "" {
				Skip("HYPERFLEET_URL is not set")
			}
			clusterName := os.Getenv("CLUSTER_NAME")
			if clusterName == "" {
				clusterName = fmt.Sprintf("hf-e2e-%d", time.Now().Unix())
			}
			Expect(len(clusterName)).To(BeNumerically("<=", hfMaxClusterNameLength),
				"CLUSTER_NAME must be ≤%d chars for Platform API", hfMaxClusterNameLength)
			rolesPrefix := os.Getenv("OPERATOR_ROLES_PREFIX")
			if rolesPrefix == "" {
				rolesPrefix = clusterName
			}

			region, err := hyperfleet.ExtractRegion(hfURL)
			if err != nil {
				envRegion := os.Getenv("AWS_DEFAULT_REGION")
				Expect(envRegion).NotTo(BeEmpty(),
					"cannot derive region from HYPERFLEET_URL; set AWS_DEFAULT_REGION")
				region = envRegion
			}

			GinkgoT().Setenv("AWS_DEFAULT_REGION", region)

			deferTeardown := func(fn func(ctx SpecContext)) {
				DeferCleanup(fn, GracePeriod(teardownGracePeriod))
			}

			By("Logging in with Platform API URL")
			_, err = rosacli.NewClient().Runner.
				Cmd("login").
				CmdFlags("--hyperfleet-url", hfURL).
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa login --hyperfleet-url")

			deferTeardown(func(ctx SpecContext) {
				By("Cleanup: removing CLI config")
				_, _ = rosacli.NewClient().Runner.Cmd("logout").Run()
			})

			By("Verifying whoami shows V2 API URL and correct region")
			whoamiRunner := rosacli.NewClient().Runner
			whoamiRunner.JsonFormat()
			whoamiOut, err := whoamiRunner.Cmd("whoami").Run()
			Expect(err).NotTo(HaveOccurred(), "rosa whoami")
			var whoamiMap map[string]interface{}
			Expect(json.Unmarshal(whoamiOut.Bytes(), &whoamiMap)).To(Succeed(),
				"parsing whoami JSON output")
			Expect(whoamiMap["V2 API"]).To(Equal(hfURL),
				"whoami must report the V2 API URL stored during login")
			Expect(whoamiMap["AWS Default Region"]).To(Equal(region),
				"whoami must report the region derived from the Platform API URL")

			By("Loading AWS configuration")
			awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
			Expect(err).NotTo(HaveOccurred(), "loading AWS config")

			By("Resolving AWS caller identity")
			stsClient := stssvc.NewFromConfig(awsCfg)
			identity, err := stsClient.GetCallerIdentity(ctx, &stssvc.GetCallerIdentityInput{})
			Expect(err).NotTo(HaveOccurred(), "STS GetCallerIdentity")
			accountID := awssdk.ToString(identity.Account)
			callerARN := awssdk.ToString(identity.Arn)

			By("Building the hyperfleet client")
			hfClient, err := hyperfleetclientset.NewForConfig(&hfrest.Config{
				Host:      hfURL,
				Region:    region,
				AccountID: accountID,
				CallerARN: callerARN,
				AWSConfig: awsCfg,
			})
			Expect(err).NotTo(HaveOccurred(), "building hyperfleet client")

			ec2Client := ec2svc.NewFromConfig(awsCfg)
			r53Client := route53svc.NewFromConfig(awsCfg)
			elbClient := elbsvc.NewFromConfig(awsCfg)
			elbv2Client := elbv2svc.NewFromConfig(awsCfg)

			const vpcCIDR = "10.0.0.0/16"
			az := region + "a"

			By("Creating VPC")
			vpcOut, err := ec2Client.CreateVpc(ctx, &ec2svc.CreateVpcInput{
				CidrBlock: awssdk.String(vpcCIDR),
				TagSpecifications: []ec2types.TagSpecification{{
					ResourceType: ec2types.ResourceTypeVpc,
					Tags: []ec2types.Tag{
						{Key: awssdk.String("Name"), Value: awssdk.String(clusterName + "-vpc")},
						{Key: awssdk.String("Cluster"), Value: awssdk.String(clusterName)},
					},
				}},
			})
			Expect(err).NotTo(HaveOccurred(), "creating VPC")
			vpcID := awssdk.ToString(vpcOut.Vpc.VpcId)
			deferTeardown(func(ctx SpecContext) {
				By("Cleanup: deleting VPC")
				if _, err := ec2Client.DeleteVpc(ctx, &ec2svc.DeleteVpcInput{VpcId: awssdk.String(vpcID)}); err != nil {
					GinkgoWriter.Printf("Failed to delete VPC %s: %v\n", vpcID, err)
					return
				}
				hfWaitVPCDeleted(ctx, ec2Client, vpcID, 2*time.Minute)
			})

			By("Waiting for VPC to become available")
			Expect(ec2svc.NewVpcAvailableWaiter(ec2Client).Wait(
				ctx,
				&ec2svc.DescribeVpcsInput{VpcIds: []string{vpcID}},
				hfVPCReadyTimeout,
			)).To(Succeed(), "waiting for VPC %s to become available", vpcID)

			By("Enabling DNS hostnames on VPC")
			_, err = ec2Client.ModifyVpcAttribute(ctx, &ec2svc.ModifyVpcAttributeInput{
				VpcId:              awssdk.String(vpcID),
				EnableDnsHostnames: &ec2types.AttributeBooleanValue{Value: awssdk.Bool(true)},
			})
			Expect(err).NotTo(HaveOccurred(), "enabling DNS hostnames on VPC %s", vpcID)
			_, err = ec2Client.ModifyVpcAttribute(ctx, &ec2svc.ModifyVpcAttributeInput{
				VpcId:            awssdk.String(vpcID),
				EnableDnsSupport: &ec2types.AttributeBooleanValue{Value: awssdk.Bool(true)},
			})
			Expect(err).NotTo(HaveOccurred(), "enabling DNS support on VPC %s", vpcID)

			By("Creating private subnet")
			privateSubnetOut, err := ec2Client.CreateSubnet(ctx, &ec2svc.CreateSubnetInput{
				VpcId:            awssdk.String(vpcID),
				CidrBlock:        awssdk.String("10.0.0.0/19"),
				AvailabilityZone: awssdk.String(az),
				TagSpecifications: []ec2types.TagSpecification{{
					ResourceType: ec2types.ResourceTypeSubnet,
					Tags: []ec2types.Tag{
						{Key: awssdk.String("Name"), Value: awssdk.String(clusterName + "-private-subnet")},
						{Key: awssdk.String("kubernetes.io/role/internal-elb"), Value: awssdk.String("1")},
						{Key: awssdk.String("Cluster"), Value: awssdk.String(clusterName)},
					},
				}},
			})
			Expect(err).NotTo(HaveOccurred(), "creating private subnet")
			subnetID := awssdk.ToString(privateSubnetOut.Subnet.SubnetId)
			deferTeardown(func(ctx SpecContext) {
				By("Cleanup: deleting private subnet")
				_, _ = ec2Client.DeleteSubnet(ctx, &ec2svc.DeleteSubnetInput{SubnetId: awssdk.String(subnetID)})
			})

			By("Creating public subnet")
			publicSubnetOut, err := ec2Client.CreateSubnet(ctx, &ec2svc.CreateSubnetInput{
				VpcId:            awssdk.String(vpcID),
				CidrBlock:        awssdk.String("10.0.101.0/24"),
				AvailabilityZone: awssdk.String(az),
				TagSpecifications: []ec2types.TagSpecification{{
					ResourceType: ec2types.ResourceTypeSubnet,
					Tags: []ec2types.Tag{
						{Key: awssdk.String("Name"), Value: awssdk.String(clusterName + "-public-subnet")},
						{Key: awssdk.String("kubernetes.io/role/elb"), Value: awssdk.String("1")},
						{Key: awssdk.String("Cluster"), Value: awssdk.String(clusterName)},
					},
				}},
			})
			Expect(err).NotTo(HaveOccurred(), "creating public subnet")
			publicSubnetID := awssdk.ToString(publicSubnetOut.Subnet.SubnetId)
			deferTeardown(func(ctx SpecContext) {
				By("Cleanup: deleting public subnet")
				_, _ = ec2Client.DeleteSubnet(ctx, &ec2svc.DeleteSubnetInput{SubnetId: awssdk.String(publicSubnetID)})
			})

			By("Waiting for subnets to become available")
			Expect(ec2svc.NewSubnetAvailableWaiter(ec2Client).Wait(
				ctx,
				&ec2svc.DescribeSubnetsInput{SubnetIds: []string{subnetID, publicSubnetID}},
				hfSubnetReadyTimeout,
			)).To(Succeed(), "waiting for subnets to become available")

			By("Creating Internet Gateway")
			igwOut, err := ec2Client.CreateInternetGateway(ctx, &ec2svc.CreateInternetGatewayInput{
				TagSpecifications: []ec2types.TagSpecification{{
					ResourceType: ec2types.ResourceTypeInternetGateway,
					Tags: []ec2types.Tag{
						{Key: awssdk.String("Name"), Value: awssdk.String(clusterName + "-igw")},
						{Key: awssdk.String("Cluster"), Value: awssdk.String(clusterName)},
					},
				}},
			})
			Expect(err).NotTo(HaveOccurred(), "creating Internet Gateway")
			igwID := awssdk.ToString(igwOut.InternetGateway.InternetGatewayId)
			igwAttached := false
			deferTeardown(func(ctx SpecContext) {
				By("Cleanup: detaching and deleting Internet Gateway")
				if igwAttached {
					_, _ = ec2Client.DetachInternetGateway(ctx, &ec2svc.DetachInternetGatewayInput{
						InternetGatewayId: awssdk.String(igwID),
						VpcId:             awssdk.String(vpcID),
					})
				}
				_, _ = ec2Client.DeleteInternetGateway(ctx, &ec2svc.DeleteInternetGatewayInput{
					InternetGatewayId: awssdk.String(igwID),
				})
			})
			_, err = ec2Client.AttachInternetGateway(ctx, &ec2svc.AttachInternetGatewayInput{
				InternetGatewayId: awssdk.String(igwID),
				VpcId:             awssdk.String(vpcID),
			})
			Expect(err).NotTo(HaveOccurred(), "attaching IGW %s to VPC %s", igwID, vpcID)
			igwAttached = true

			By("Allocating Elastic IP for NAT Gateway")
			eipOut, err := ec2Client.AllocateAddress(ctx, &ec2svc.AllocateAddressInput{
				Domain: ec2types.DomainTypeVpc,
				TagSpecifications: []ec2types.TagSpecification{{
					ResourceType: ec2types.ResourceTypeElasticIp,
					Tags: []ec2types.Tag{
						{Key: awssdk.String("Name"), Value: awssdk.String(clusterName + "-nat-eip")},
						{Key: awssdk.String("Cluster"), Value: awssdk.String(clusterName)},
					},
				}},
			})
			Expect(err).NotTo(HaveOccurred(), "allocating EIP for NAT Gateway")
			natEIPAllocID := awssdk.ToString(eipOut.AllocationId)
			deferTeardown(func(ctx SpecContext) {
				By("Cleanup: releasing NAT Gateway EIP")
				_, _ = ec2Client.ReleaseAddress(ctx, &ec2svc.ReleaseAddressInput{
					AllocationId: awssdk.String(natEIPAllocID),
				})
			})

			By("Creating NAT Gateway")
			natOut, err := ec2Client.CreateNatGateway(ctx, &ec2svc.CreateNatGatewayInput{
				SubnetId:         awssdk.String(publicSubnetID),
				AllocationId:     awssdk.String(natEIPAllocID),
				ConnectivityType: ec2types.ConnectivityTypePublic,
				TagSpecifications: []ec2types.TagSpecification{{
					ResourceType: ec2types.ResourceTypeNatgateway,
					Tags: []ec2types.Tag{
						{Key: awssdk.String("Name"), Value: awssdk.String(clusterName + "-natgw")},
						{Key: awssdk.String("Cluster"), Value: awssdk.String(clusterName)},
					},
				}},
			})
			Expect(err).NotTo(HaveOccurred(), "creating NAT Gateway")
			natGWID := awssdk.ToString(natOut.NatGateway.NatGatewayId)
			deferTeardown(func(ctx SpecContext) {
				By("Cleanup: deleting NAT Gateway")
				_, _ = ec2Client.DeleteNatGateway(ctx, &ec2svc.DeleteNatGatewayInput{
					NatGatewayId: awssdk.String(natGWID),
				})
				_ = ec2svc.NewNatGatewayDeletedWaiter(ec2Client).Wait(
					ctx,
					&ec2svc.DescribeNatGatewaysInput{NatGatewayIds: []string{natGWID}},
					5*time.Minute,
				)
			})

			By("Waiting for NAT Gateway to become available")
			Expect(ec2svc.NewNatGatewayAvailableWaiter(ec2Client).Wait(
				ctx,
				&ec2svc.DescribeNatGatewaysInput{NatGatewayIds: []string{natGWID}},
				5*time.Minute,
			)).To(Succeed(), "waiting for NAT Gateway %s to become available", natGWID)

			By("Creating public route table with Internet Gateway route")
			pubRTOut, err := ec2Client.CreateRouteTable(ctx, &ec2svc.CreateRouteTableInput{
				VpcId: awssdk.String(vpcID),
				TagSpecifications: []ec2types.TagSpecification{{
					ResourceType: ec2types.ResourceTypeRouteTable,
					Tags: []ec2types.Tag{
						{Key: awssdk.String("Name"), Value: awssdk.String(clusterName + "-public-rtb")},
						{Key: awssdk.String("Cluster"), Value: awssdk.String(clusterName)},
					},
				}},
			})
			Expect(err).NotTo(HaveOccurred(), "creating public route table")
			publicRTID := awssdk.ToString(pubRTOut.RouteTable.RouteTableId)
			var publicRTAssocID string
			deferTeardown(func(ctx SpecContext) {
				By("Cleanup: deleting public route table")
				if publicRTAssocID != "" {
					_, _ = ec2Client.DisassociateRouteTable(ctx, &ec2svc.DisassociateRouteTableInput{
						AssociationId: awssdk.String(publicRTAssocID),
					})
				}
				_, _ = ec2Client.DeleteRouteTable(ctx, &ec2svc.DeleteRouteTableInput{
					RouteTableId: awssdk.String(publicRTID),
				})
			})
			_, err = ec2Client.CreateRoute(ctx, &ec2svc.CreateRouteInput{
				RouteTableId:         awssdk.String(publicRTID),
				DestinationCidrBlock: awssdk.String("0.0.0.0/0"),
				GatewayId:            awssdk.String(igwID),
			})
			Expect(err).NotTo(HaveOccurred(), "adding IGW route to public route table")
			pubAssocOut, err := ec2Client.AssociateRouteTable(ctx, &ec2svc.AssociateRouteTableInput{
				RouteTableId: awssdk.String(publicRTID),
				SubnetId:     awssdk.String(publicSubnetID),
			})
			Expect(err).NotTo(HaveOccurred(), "associating public subnet with public route table")
			publicRTAssocID = awssdk.ToString(pubAssocOut.AssociationId)

			By("Creating private route table with NAT Gateway route")
			privRTOut, err := ec2Client.CreateRouteTable(ctx, &ec2svc.CreateRouteTableInput{
				VpcId: awssdk.String(vpcID),
				TagSpecifications: []ec2types.TagSpecification{{
					ResourceType: ec2types.ResourceTypeRouteTable,
					Tags: []ec2types.Tag{
						{Key: awssdk.String("Name"), Value: awssdk.String(clusterName + "-private-rtb")},
						{Key: awssdk.String("Cluster"), Value: awssdk.String(clusterName)},
					},
				}},
			})
			Expect(err).NotTo(HaveOccurred(), "creating private route table")
			privateRTID := awssdk.ToString(privRTOut.RouteTable.RouteTableId)
			var privateRTAssocID string
			deferTeardown(func(ctx SpecContext) {
				By("Cleanup: deleting private route table")
				if privateRTAssocID != "" {
					_, _ = ec2Client.DisassociateRouteTable(ctx, &ec2svc.DisassociateRouteTableInput{
						AssociationId: awssdk.String(privateRTAssocID),
					})
				}
				_, _ = ec2Client.DeleteRouteTable(ctx, &ec2svc.DeleteRouteTableInput{
					RouteTableId: awssdk.String(privateRTID),
				})
			})
			_, err = ec2Client.CreateRoute(ctx, &ec2svc.CreateRouteInput{
				RouteTableId:         awssdk.String(privateRTID),
				DestinationCidrBlock: awssdk.String("0.0.0.0/0"),
				NatGatewayId:         awssdk.String(natGWID),
			})
			Expect(err).NotTo(HaveOccurred(), "adding NAT Gateway route to private route table")
			privAssocOut, err := ec2Client.AssociateRouteTable(ctx, &ec2svc.AssociateRouteTableInput{
				RouteTableId: awssdk.String(privateRTID),
				SubnetId:     awssdk.String(subnetID),
			})
			Expect(err).NotTo(HaveOccurred(), "associating private subnet with private route table")
			privateRTAssocID = awssdk.ToString(privAssocOut.AssociationId)

			By("Creating worker security group")
			sgOut, err := ec2Client.CreateSecurityGroup(ctx, &ec2svc.CreateSecurityGroupInput{
				GroupName:   awssdk.String(clusterName + "-hc-worker-sg"),
				Description: awssdk.String("Worker node security group for " + clusterName),
				VpcId:       awssdk.String(vpcID),
				TagSpecifications: []ec2types.TagSpecification{{
					ResourceType: ec2types.ResourceTypeSecurityGroup,
					Tags: []ec2types.Tag{
						{Key: awssdk.String("Name"), Value: awssdk.String(clusterName + "-hc-worker-sg")},
						{Key: awssdk.String("Cluster"), Value: awssdk.String(clusterName)},
					},
				}},
			})
			Expect(err).NotTo(HaveOccurred(), "creating worker security group")
			workerSGID := awssdk.ToString(sgOut.GroupId)
			deferTeardown(func(ctx SpecContext) {
				By("Cleanup: deleting worker security group")
				_, _ = ec2Client.DeleteSecurityGroup(ctx, &ec2svc.DeleteSecurityGroupInput{
					GroupId: awssdk.String(workerSGID),
				})
			})
			_, err = ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2svc.AuthorizeSecurityGroupIngressInput{
				GroupId: awssdk.String(workerSGID),
				IpPermissions: []ec2types.IpPermission{
					{
						IpProtocol: awssdk.String("-1"),
						UserIdGroupPairs: []ec2types.UserIdGroupPair{
							{GroupId: awssdk.String(workerSGID)},
						},
					},
					{
						IpProtocol: awssdk.String("-1"),
						IpRanges:   []ec2types.IpRange{{CidrIp: awssdk.String(vpcCIDR)}},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred(), "adding ingress rules to worker security group")

			By("Creating private hosted zone for PrivateLink DNS")
			hzOut, err := r53Client.CreateHostedZone(ctx, &route53svc.CreateHostedZoneInput{
				Name:             awssdk.String(clusterName + ".hypershift.local"),
				CallerReference:  awssdk.String(fmt.Sprintf("%s-%d", clusterName, time.Now().UnixNano())),
				HostedZoneConfig: &route53types.HostedZoneConfig{PrivateZone: true},
				VPC: &route53types.VPC{
					VPCId:     awssdk.String(vpcID),
					VPCRegion: route53types.VPCRegion(region),
				},
			})
			Expect(err).NotTo(HaveOccurred(), "creating private hosted zone %s.hypershift.local", clusterName)
			hostedZoneID := awssdk.ToString(hzOut.HostedZone.Id)
			hostedZoneIDShort := strings.TrimPrefix(hostedZoneID, "/hostedzone/")
			_, err = r53Client.ChangeTagsForResource(ctx, &route53svc.ChangeTagsForResourceInput{
				ResourceType: route53types.TagResourceTypeHostedzone,
				ResourceId:   awssdk.String(hostedZoneIDShort),
				AddTags: []route53types.Tag{
					{Key: awssdk.String("Name"), Value: awssdk.String(clusterName + ".hypershift.local")},
					{Key: awssdk.String(fmt.Sprintf("kubernetes.io/cluster/%s", clusterName)), Value: awssdk.String("owned")},
					{Key: awssdk.String("Cluster"), Value: awssdk.String(clusterName)},
					{Key: awssdk.String("ManagedBy"), Value: awssdk.String("rosactl")},
				},
			})
			Expect(err).NotTo(HaveOccurred(), "tagging hosted zone %s", hostedZoneID)
			GinkgoWriter.Printf("Private hosted zone %s.hypershift.local created: %s\n", clusterName, hostedZoneID)
			deferTeardown(func(ctx SpecContext) {
				By("Cleanup: purging records and deleting private hosted zone")
				hfPurgeHostedZoneRecords(ctx, r53Client, hostedZoneIDShort)
				_, _ = r53Client.DeleteHostedZone(ctx, &route53svc.DeleteHostedZoneInput{
					Id: awssdk.String(hostedZoneID),
				})
			})

			oidcRunner := rosacli.NewClient().Runner

			By("Creating managed OIDC config via CLI")
			oidcCreateOut, err := oidcRunner.JsonFormat().Cmd("create", "oidc-config").
				CmdFlags("--managed", "--mode", "auto", "-y").
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa create oidc-config")

			parsed := rosacli.NewParser().JsonData.Input(oidcCreateOut).Parse()
			oidcConfigID := parsed.DigString("id")
			if oidcConfigID == "" {
				oidcConfigID = parsed.DigString("metadata", "name")
			}
			Expect(oidcConfigID).NotTo(BeEmpty(), "OIDC config ID must be returned by create")
			GinkgoWriter.Printf("OIDC config created with ID %s\n", oidcConfigID)

			issuerURL := parsed.DigString("spec", "issuerUrl")
			if issuerURL == "" {
				issuerURL = parsed.DigString("spec", "issuer_url")
			}
			Expect(issuerURL).NotTo(BeEmpty(), "OIDC issuer URL must be returned by create")

			By("Verifying IAM OIDC provider exists after managed OIDC create")
			parsedIssuerURL, err := urlHelper.ParseRequestURI(issuerURL)
			Expect(err).NotTo(HaveOccurred(), "parsing OIDC issuer URL")
			providerURL := fmt.Sprintf("%s%s", parsedIssuerURL.Host, parsedIssuerURL.Path)
			callerParsedARN, err := arn.Parse(callerARN)
			Expect(err).NotTo(HaveOccurred(), "parsing caller ARN")
			oidcProviderARN := rosaaws.GetOIDCProviderARN(
				callerParsedARN.Partition, accountID, providerURL)
			providerOut, err := iamsvc.NewFromConfig(awsCfg).GetOpenIDConnectProvider(
				ctx, &iamsvc.GetOpenIDConnectProviderInput{
					OpenIDConnectProviderArn: awssdk.String(oidcProviderARN),
				})
			Expect(err).NotTo(HaveOccurred(), "IAM OIDC provider must exist after managed OIDC create")
			Expect(awssdk.ToString(providerOut.Url)).To(Equal(providerURL))

			deferTeardown(func(ctx SpecContext) {
				By("Cleanup: initiating OIDC config deletion (async, no list wait)")
				hfInitiateOidcConfigDelete(oidcConfigID)
			})

			By("Listing OIDC configs via CLI")
			listOidcOut, err := oidcRunner.Cmd("list", "oidc-config").CmdFlags().Run()
			Expect(err).NotTo(HaveOccurred(), "rosa list oidc-config")
			Expect(listOidcOut.String()).To(ContainSubstring(oidcConfigID),
				"created OIDC config must appear in list output")

			By("Creating operator roles via CLI")
			oidcRunner.UnsetFormat()
			_, err = oidcRunner.Cmd("create", "operator-roles").
				CmdFlags(
					"--hosted-cp",
					"--prefix", rolesPrefix,
					"--oidc-config-id", oidcConfigID,
					"--mode", "auto",
					"-y",
				).
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa create operator-roles")

			deferTeardown(func(ctx SpecContext) {
				By("Cleanup: initiating operator roles deletion (async)")
				hfInitiateOperatorRolesDelete(rolesPrefix)
			})

			By("Creating cluster via CLI")
			version := os.Getenv("HYPERFLEET_VERSION")
			createArgs := []string{
				"--cluster-name", clusterName,
				"--subnet-ids", subnetID,
				"--operator-roles-prefix", rolesPrefix,
				"--oidc-config-id", oidcConfigID,
			}
			if version != "" {
				createArgs = append(createArgs, "--version", version)
			}
			_, err = rosacli.NewClient().Runner.
				Cmd("create", "cluster").
				CmdFlags(createArgs...).
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa create cluster CLI call")

			var clusterID string
			deferTeardown(func(ctx SpecContext) {
				By("Cleanup: deleting cluster via CLI")
				_, _ = rosacli.NewClient().Runner.
					Cmd("delete", "cluster").
					CmdFlags("-c", clusterName, "-y").
					Run()

				if clusterID == "" {
					return
				}
				By("Waiting for cluster to be fully deleted before releasing AWS resources")
				_ = hfClient.HyperfleetV1alpha1().Clusters().WaitUntil(
					ctx, clusterID,
					func(c *v1alpha1.Cluster) bool {
						if c == nil {
							GinkgoWriter.Printf("Cluster %s deleted\n", clusterName)
							return true
						}
						GinkgoWriter.Printf("[%s] cluster %s: phase=%s, waiting for deletion\n",
							time.Now().Format(time.RFC3339), clusterName, c.Status.Phase)
						return false
					},
					hfClusterReadyInterval, hfClusterReadyTimeout,
				)

				By("Waiting for worker EC2 instances in VPC to terminate")
				hfWaitVPCInstancesTerminated(ctx, ec2Client, vpcID, 15*time.Minute)

				By("Deleting classic load balancers created by cluster ingress controller")
				hfDeleteVPCClassicLoadBalancers(ctx, elbClient, vpcID)
				hfWaitVPCClassicLoadBalancersDeleted(ctx, elbClient, vpcID, 5*time.Minute)

				By("Deleting ALBs/NLBs created by cluster ingress controller")
				hfDeleteVPCLoadBalancers(ctx, elbv2Client, vpcID)
				hfWaitVPCLoadBalancersDeleted(ctx, elbv2Client, vpcID, 5*time.Minute)

				By("Releasing orphaned ENIs left by cluster controllers")
				hfDeleteAvailableENIs(ctx, ec2Client, vpcID)

				By("Deleting non-default security groups left by cluster controllers")
				hfDeleteVPCSecurityGroups(ctx, ec2Client, vpcID)
			})

			By("Fetching cluster ID via CLI describe")
			describeCreateRunner := rosacli.NewClient().Runner
			describeCreateRunner.JsonFormat()
			describeCreateOut, err := describeCreateRunner.Cmd("describe", "cluster").
				CmdFlags("-c", clusterName).
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa describe cluster after create")

			var createDescribeMap map[string]interface{}
			Expect(json.Unmarshal(describeCreateOut.Bytes(), &createDescribeMap)).To(Succeed(),
				"parsing describe JSON output after create")
			clusterID, _ = createDescribeMap["id"].(string)
			Expect(clusterID).NotTo(BeEmpty(), "cluster UID must be returned by describe after create")
			GinkgoWriter.Printf("Cluster %q created with ID %s\n", clusterName, clusterID)

			specMap, _ := createDescribeMap["spec"].(map[string]interface{})
			issuerURL, _ = specMap["oidc_issuer"].(string)
			Expect(issuerURL).NotTo(BeEmpty(), "OIDC IssuerURL must be present in describe response after create")
			GinkgoWriter.Printf("OIDC IssuerURL: %s\n", issuerURL)

			By("Waiting for cluster to become Ready")
			var clusterPhase v1alpha1.ClusterPhase
			err = hfClient.HyperfleetV1alpha1().Clusters().WaitUntil(
				ctx,
				clusterID,
				func(c *v1alpha1.Cluster) bool {
					if c == nil {
						return false
					}
					clusterPhase = c.Status.Phase
					if clusterPhase != "" {
						GinkgoWriter.Printf("Cluster %q phase: %s\n", clusterName, clusterPhase)
					}
					return clusterPhase == v1alpha1.ClusterPhaseReady
				},
				hfClusterReadyInterval,
				hfClusterReadyTimeout,
			)
			Expect(err).NotTo(HaveOccurred(), "waiting for cluster to become Ready")
			Expect(clusterPhase).To(Equal(v1alpha1.ClusterPhaseReady), "cluster must reach Ready phase")

			By("Listing clusters via CLI and verifying the new cluster appears")
			rosaRunner := rosacli.NewClient().Runner
			listOut, err := rosaRunner.Cmd("list", "clusters").Run()
			Expect(err).NotTo(HaveOccurred(), "rosa list clusters CLI call")
			Expect(listOut.String()).To(ContainSubstring(clusterID),
				"cluster UID must appear in rosa list clusters output")
			Expect(listOut.String()).To(ContainSubstring(clusterName),
				"cluster name must appear in rosa list clusters output")

			By("Describing cluster via CLI and comparing with Get response")
			getOut, err := hfClient.HyperfleetV1alpha1().Clusters().Get(
				ctx, clusterID, platform.GetOptions{},
			)
			Expect(err).NotTo(HaveOccurred(), "SDK Get before CLI describe")

			describeRunner := rosacli.NewClient().Runner
			describeRunner.JsonFormat()
			cliOut, err := describeRunner.Cmd("describe", "cluster").
				CmdFlags("-c", clusterName).
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa describe cluster CLI call")
			describeRunner.UnsetFormat()

			var describeMap map[string]interface{}
			Expect(json.Unmarshal(cliOut.Bytes(), &describeMap)).To(Succeed(),
				"parsing describe JSON output")

			Expect(describeMap["id"]).To(Equal(clusterID),
				"CLI describe id must match cluster UID")
			Expect(describeMap["name"]).To(Equal(clusterName),
				"CLI describe name must match cluster name")
			Expect(describeMap["state"]).To(Equal(string(v1alpha1.ClusterPhaseReady)),
				"CLI describe state must be Ready")
			Expect(describeMap["api_url"]).To(Equal(
				fmt.Sprintf("https://%s:%d",
					getOut.Status.ControlPlaneEndpoint.Host,
					getOut.Status.ControlPlaneEndpoint.Port),
			),
				"CLI describe api_url must match Get control plane endpoint")

			GinkgoWriter.Printf("rosa describe cluster output:\n%s\n", cliOut.String())

			instanceType := os.Getenv("HYPERFLEET_INSTANCE_TYPE")
			if instanceType == "" {
				instanceType = hfDefaultInstanceType
			}
			const np1Name = "np1"
			const np2Name = "np2"
			var np1ID, np2ID string

			deferTeardown(func(ctx SpecContext) {
				By("Cleanup: initiating node pool 1 deletion")
				_, _ = rosacli.NewClient().Runner.
					Cmd("delete", "machinepool").
					CmdFlags("-c", clusterName, "--machinepool", np1Name, "--yes").
					Run()
			})

			By("Creating first node pool via CLI")
			_, err = rosacli.NewClient().Runner.
				Cmd("create", "machinepool").
				CmdFlags("-c", clusterName,
					"--name", np1Name,
					"--replicas", "2",
					"--instance-type", instanceType,
					"--subnet", subnetID).
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa create machinepool %s", np1Name)

			By("Creating second node pool via CLI")
			_, err = rosacli.NewClient().Runner.
				Cmd("create", "machinepool").
				CmdFlags("-c", clusterName,
					"--name", np2Name,
					"--replicas", "1",
					"--instance-type", instanceType,
					"--subnet", subnetID).
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa create machinepool %s", np2Name)

			By("Resolving node pool IDs via API")
			npList, err := hfClient.HyperfleetV1alpha1().NodePools(clusterID).List(ctx, platform.ListOptions{})
			Expect(err).NotTo(HaveOccurred(), "listing node pools")
			for i := range npList.Items {
				switch npList.Items[i].Name {
				case np1Name:
					np1ID = string(npList.Items[i].UID)
				case np2Name:
					np2ID = string(npList.Items[i].UID)
				}
			}
			Expect(np1ID).NotTo(BeEmpty(), "node pool %s not found in list", np1Name)
			Expect(np2ID).NotTo(BeEmpty(), "node pool %s not found in list", np2Name)
			GinkgoWriter.Printf("NodePool %s id=%s, NodePool %s id=%s\n", np1Name, np1ID, np2Name, np2ID)

			nodePools := hfClient.HyperfleetV1alpha1().NodePools(clusterID)

			By("Waiting for node pool 1 to become Ready")
			Expect(nodePools.WaitUntil(ctx, np1ID,
				func(n *v1alpha1.NodePool) bool {
					if n == nil {
						return false
					}
					GinkgoWriter.Printf("[%s] node pool %s: phase=%s conditions=%s\n",
						time.Now().Format(time.RFC3339), np1Name, n.Status.Phase,
						hfFormatNodePoolConditions(n.Status.Conditions))
					return n.Status.Phase == v1alpha1.NodePoolPhaseReady
				},
				hfNodePoolReadyInterval, hfNodePoolReadyTimeout,
			)).To(Succeed(), "node pool %s should reach Ready phase", np1Name)
			GinkgoWriter.Printf("NodePool %s is Ready\n", np1Name)

			By("Waiting for node pool 2 to become Ready")
			Expect(nodePools.WaitUntil(ctx, np2ID,
				func(n *v1alpha1.NodePool) bool {
					if n == nil {
						return false
					}
					GinkgoWriter.Printf("[%s] node pool %s: phase=%s conditions=%s\n",
						time.Now().Format(time.RFC3339), np2Name, n.Status.Phase,
						hfFormatNodePoolConditions(n.Status.Conditions))
					return n.Status.Phase == v1alpha1.NodePoolPhaseReady
				},
				hfNodePoolReadyInterval, hfNodePoolReadyTimeout,
			)).To(Succeed(), "node pool %s should reach Ready phase", np2Name)
			GinkgoWriter.Printf("NodePool %s is Ready\n", np2Name)

			By("Listing machine pools via CLI and verifying both appear")
			listMpOut, err := rosacli.NewClient().Runner.
				Cmd("list", "machinepool").
				CmdFlags("-c", clusterName).
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa list machinepool")
			Expect(listMpOut.String()).To(ContainSubstring(np1Name),
				"node pool 1 must appear in rosa list machinepool output")
			Expect(listMpOut.String()).To(ContainSubstring(np2Name),
				"node pool 2 must appear in rosa list machinepool output")

			By("Describing node pool 1 via CLI")
			describeNp1Runner := rosacli.NewClient().Runner
			describeNp1Runner.JsonFormat()
			np1Out, err := describeNp1Runner.Cmd("describe", "machinepool").
				CmdFlags("-c", clusterName, "--machinepool", np1Name).
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa describe machinepool %s", np1Name)
			var np1Map map[string]interface{}
			Expect(json.Unmarshal(np1Out.Bytes(), &np1Map)).To(Succeed(),
				"parsing describe machinepool %s JSON", np1Name)
			Expect(np1Map["id"]).To(Equal(np1ID),
				"CLI describe machinepool id must match node pool UID")
			GinkgoWriter.Printf("rosa describe machinepool %s output:\n%s\n", np1Name, np1Out.String())

			By("Scaling node pool 1 replicas from 2 to 3 via CLI edit")
			_, err = rosacli.NewClient().Runner.
				Cmd("edit", "machinepool", np1Name).
				CmdFlags("-c", clusterName, "--replicas", "3").
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa edit machinepool %s --replicas 3", np1Name)

			By("Verifying node pool 1 replica count via CLI describe")
			np1EditOut, err := rosacli.NewClient().Runner.
				Cmd("describe", "machinepool").
				CmdFlags("-c", clusterName, "--machinepool", np1Name, "-o", "json").
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa describe machinepool %s after edit", np1Name)
			var np1EditMap map[string]interface{}
			Expect(json.Unmarshal(np1EditOut.Bytes(), &np1EditMap)).To(Succeed(),
				"parsing describe machinepool %s JSON after edit", np1Name)
			Expect(np1EditMap["replicas"]).To(BeEquivalentTo(3),
				"node pool %s replicas must be 3 after edit", np1Name)
			GinkgoWriter.Printf("NodePool %s after edit:\n%s\n", np1Name, np1EditOut.String())

			By("Initiating node pool 2 deletion via CLI (cluster delete completes it)")
			_, err = rosacli.NewClient().Runner.
				Cmd("delete", "machinepool").
				CmdFlags("-c", clusterName, "--machinepool", np2Name, "--yes").
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa delete machinepool %s", np2Name)
			GinkgoWriter.Printf("NodePool %s delete initiated; not waiting for operator finalizers\n", np2Name)

			By("Initiating node pool 1 deletion (cluster delete will complete it)")
			_, err = rosacli.NewClient().Runner.
				Cmd("delete", "machinepool").
				CmdFlags("-c", clusterName, "--machinepool", np1Name, "--yes").
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa delete machinepool %s", np1Name)
		})
	},
)

// hfInitiateOidcConfigDelete fires async OIDC teardown; do not list immediately after.
func hfInitiateOidcConfigDelete(oidcConfigID string) {
	_, _ = rosacli.NewClient().Runner.
		Cmd("delete", "oidc-config").
		CmdFlags("--oidc-config-id", oidcConfigID, "--mode", "auto", "-y").
		Run()
}

func hfInitiateOperatorRolesDelete(rolesPrefix string) {
	_, _ = rosacli.NewClient().Runner.
		Cmd("delete", "operator-roles").
		CmdFlags("--prefix", rolesPrefix, "--hosted-cp", "--mode", "auto", "-y").
		Run()
}

func hfFormatNodePoolConditions(conditions []metav1.Condition) string {
	if len(conditions) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(conditions))
	for _, c := range conditions {
		msg := c.Message
		if msg == "" {
			msg = c.Reason
		}
		parts = append(parts, fmt.Sprintf("%s=%s(%s)", c.Type, c.Status, msg))
	}
	return strings.Join(parts, "; ")
}

// hfPurgeHostedZoneRecords deletes all non-default record sets (everything
// except NS and SOA) from the hosted zone so that DeleteHostedZone succeeds.
// The operator writes A/CNAME records into the zone for ingress and the
// ignition server; if those are not removed first, zone deletion fails.
func hfPurgeHostedZoneRecords(ctx context.Context, r53Client *route53svc.Client, zoneID string) {
	out, err := r53Client.ListResourceRecordSets(ctx, &route53svc.ListResourceRecordSetsInput{
		HostedZoneId: awssdk.String(zoneID),
	})
	if err != nil {
		GinkgoWriter.Printf("ListResourceRecordSets error for zone %s: %v\n", zoneID, err)
		return
	}
	var changes []route53types.Change
	for i := range out.ResourceRecordSets {
		rrs := out.ResourceRecordSets[i]
		if rrs.Type == route53types.RRTypeNs || rrs.Type == route53types.RRTypeSoa {
			continue
		}
		changes = append(changes, route53types.Change{
			Action:            route53types.ChangeActionDelete,
			ResourceRecordSet: &rrs,
		})
	}
	if len(changes) == 0 {
		return
	}
	_, err = r53Client.ChangeResourceRecordSets(ctx, &route53svc.ChangeResourceRecordSetsInput{
		HostedZoneId: awssdk.String(zoneID),
		ChangeBatch:  &route53types.ChangeBatch{Changes: changes},
	})
	if err != nil {
		GinkgoWriter.Printf("Failed to purge records from zone %s: %v\n", zoneID, err)
	} else {
		GinkgoWriter.Printf("Purged %d record(s) from zone %s\n", len(changes), zoneID)
	}
}

// hfDeleteVPCClassicLoadBalancers deletes all classic (ELBv1) load balancers
// in the VPC. Kubernetes Services of type LoadBalancer in older clusters may
// create classic ELBs; if not removed they block VPC deletion.
func hfDeleteVPCClassicLoadBalancers(ctx context.Context, elbClient *elbsvc.Client, vpcID string) {
	var marker *string
	for {
		out, err := elbClient.DescribeLoadBalancers(ctx, &elbsvc.DescribeLoadBalancersInput{
			Marker: marker,
		})
		if err != nil {
			GinkgoWriter.Printf("DescribeLoadBalancers (classic) error: %v\n", err)
			return
		}
		for _, lb := range out.LoadBalancerDescriptions {
			if awssdk.ToString(lb.VPCId) != vpcID {
				continue
			}
			name := awssdk.ToString(lb.LoadBalancerName)
			_, delErr := elbClient.DeleteLoadBalancer(ctx, &elbsvc.DeleteLoadBalancerInput{
				LoadBalancerName: awssdk.String(name),
			})
			if delErr != nil {
				GinkgoWriter.Printf("Failed to delete classic load balancer %s: %v\n", name, delErr)
			} else {
				GinkgoWriter.Printf("Deleted classic load balancer %s\n", name)
			}
		}
		if out.NextMarker == nil {
			break
		}
		marker = out.NextMarker
	}
}

// hfDeleteVPCLoadBalancers deletes all ALBs and NLBs (ELBv2) in the VPC. The
// cluster ingress controller creates these; if they are not removed before VPC
// teardown the subnets and security groups they reference cannot be deleted.
func hfDeleteVPCLoadBalancers(ctx context.Context, elbv2Client *elbv2svc.Client, vpcID string) {
	var marker *string
	for {
		out, err := elbv2Client.DescribeLoadBalancers(ctx, &elbv2svc.DescribeLoadBalancersInput{
			Marker: marker,
		})
		if err != nil {
			GinkgoWriter.Printf("DescribeLoadBalancers error: %v\n", err)
			return
		}
		for _, lb := range out.LoadBalancers {
			if awssdk.ToString(lb.VpcId) != vpcID {
				continue
			}
			lbARN := awssdk.ToString(lb.LoadBalancerArn)
			_, delErr := elbv2Client.DeleteLoadBalancer(ctx, &elbv2svc.DeleteLoadBalancerInput{
				LoadBalancerArn: awssdk.String(lbARN),
			})
			if delErr != nil {
				GinkgoWriter.Printf("Failed to delete load balancer %s: %v\n", lbARN, delErr)
			} else {
				GinkgoWriter.Printf("Deleted load balancer %s\n", awssdk.ToString(lb.LoadBalancerName))
			}
		}
		if out.NextMarker == nil {
			break
		}
		marker = out.NextMarker
	}
}

// hfWaitVPCClassicLoadBalancersDeleted polls until no classic (ELBv1) load
// balancers remain in the VPC. Classic LB deletion is synchronous but the SG
// dependency is released only after the LB is fully gone.
func hfWaitVPCClassicLoadBalancersDeleted(ctx context.Context, elbClient *elbsvc.Client, vpcID string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := elbClient.DescribeLoadBalancers(ctx, &elbsvc.DescribeLoadBalancersInput{})
		if err != nil {
			GinkgoWriter.Printf("DescribeLoadBalancers (classic) error while waiting: %v\n", err)
			return
		}
		found := 0
		for _, lb := range out.LoadBalancerDescriptions {
			if awssdk.ToString(lb.VPCId) == vpcID {
				found++
			}
		}
		if found == 0 {
			GinkgoWriter.Printf("All classic load balancers in VPC %s deleted\n", vpcID)
			return
		}
		GinkgoWriter.Printf("[%s] %d classic load balancer(s) still present in VPC %s, waiting...\n",
			time.Now().Format(time.RFC3339), found, vpcID)
		time.Sleep(10 * time.Second)
	}
	GinkgoWriter.Printf("WARNING: timed out waiting for classic load balancers in VPC %s to be deleted\n", vpcID)
}

// hfWaitVPCLoadBalancersDeleted polls until no ALBs or NLBs (ELBv2) remain in
// the VPC. ALB/NLB deletion is asynchronous; the security group dependency is
// not released until AWS fully removes the load balancer.
func hfWaitVPCLoadBalancersDeleted(ctx context.Context, elbv2Client *elbv2svc.Client, vpcID string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := elbv2Client.DescribeLoadBalancers(ctx, &elbv2svc.DescribeLoadBalancersInput{})
		if err != nil {
			GinkgoWriter.Printf("DescribeLoadBalancers error while waiting: %v\n", err)
			return
		}
		found := 0
		for _, lb := range out.LoadBalancers {
			if awssdk.ToString(lb.VpcId) == vpcID {
				found++
			}
		}
		if found == 0 {
			GinkgoWriter.Printf("All load balancers in VPC %s deleted\n", vpcID)
			return
		}
		GinkgoWriter.Printf("[%s] %d load balancer(s) still present in VPC %s, waiting...\n",
			time.Now().Format(time.RFC3339), found, vpcID)
		time.Sleep(10 * time.Second)
	}
	GinkgoWriter.Printf("WARNING: timed out waiting for load balancers in VPC %s to be deleted\n", vpcID)
}

// hfDeleteAvailableENIs deletes all network interfaces in the VPC that are in
// the "available" state (not attached to any resource). These are typically
// ENIs left behind by cluster controllers (ingress, CSI, cloud controller)
// after the cluster is deleted, which would otherwise block VPC deletion.
func hfDeleteAvailableENIs(ctx context.Context, ec2Client *ec2svc.Client, vpcID string) {
	out, err := ec2Client.DescribeNetworkInterfaces(ctx, &ec2svc.DescribeNetworkInterfacesInput{
		Filters: []ec2types.Filter{
			{Name: awssdk.String("vpc-id"), Values: []string{vpcID}},
			{Name: awssdk.String("status"), Values: []string{"available"}},
		},
	})
	if err != nil {
		GinkgoWriter.Printf("DescribeNetworkInterfaces error for VPC %s: %v\n", vpcID, err)
		return
	}
	for _, eni := range out.NetworkInterfaces {
		eniID := awssdk.ToString(eni.NetworkInterfaceId)
		_, delErr := ec2Client.DeleteNetworkInterface(ctx, &ec2svc.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: awssdk.String(eniID),
		})
		if delErr != nil {
			GinkgoWriter.Printf("Failed to delete ENI %s: %v\n", eniID, delErr)
		} else {
			GinkgoWriter.Printf("Deleted orphaned ENI %s\n", eniID)
		}
	}
}

// hfDeleteVPCSecurityGroups deletes all non-default security groups in the VPC.
// The cluster controller and ingress controller create SGs that are not removed
// when the cluster is deleted; they block VPC deletion if left behind.
func hfDeleteVPCSecurityGroups(ctx context.Context, ec2Client *ec2svc.Client, vpcID string) {
	out, err := ec2Client.DescribeSecurityGroups(ctx, &ec2svc.DescribeSecurityGroupsInput{
		Filters: []ec2types.Filter{
			{Name: awssdk.String("vpc-id"), Values: []string{vpcID}},
		},
	})
	if err != nil {
		GinkgoWriter.Printf("DescribeSecurityGroups error for VPC %s: %v\n", vpcID, err)
		return
	}
	for _, sg := range out.SecurityGroups {
		if awssdk.ToString(sg.GroupName) == "default" {
			continue
		}
		sgID := awssdk.ToString(sg.GroupId)
		_, delErr := ec2Client.DeleteSecurityGroup(ctx, &ec2svc.DeleteSecurityGroupInput{
			GroupId: awssdk.String(sgID),
		})
		if delErr != nil {
			GinkgoWriter.Printf("Failed to delete security group %s: %v\n", sgID, delErr)
		} else {
			GinkgoWriter.Printf("Deleted security group %s (%s)\n", sgID, awssdk.ToString(sg.GroupName))
		}
	}
}

// hfWaitVPCDeleted polls DescribeVpcs until the VPC is no longer present or
// the timeout expires. Logs a warning on timeout rather than failing so that a
// cleanup race does not mask the actual test result.
func hfWaitVPCDeleted(ctx context.Context, ec2Client *ec2svc.Client, vpcID string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := ec2Client.DescribeVpcs(ctx, &ec2svc.DescribeVpcsInput{
			Filters: []ec2types.Filter{
				{Name: awssdk.String("vpc-id"), Values: []string{vpcID}},
			},
		})
		if err != nil {
			GinkgoWriter.Printf("DescribeVpcs error while waiting for VPC %s deletion: %v\n", vpcID, err)
			return
		}
		if len(out.Vpcs) == 0 {
			GinkgoWriter.Printf("VPC %s deleted\n", vpcID)
			return
		}
		GinkgoWriter.Printf("[%s] VPC %s still present (state: %s), waiting...\n",
			time.Now().Format(time.RFC3339), vpcID, out.Vpcs[0].State)
		time.Sleep(10 * time.Second)
	}
	GinkgoWriter.Printf("WARNING: timed out waiting for VPC %s to be deleted\n", vpcID)
}

// hfWaitVPCInstancesTerminated polls until no non-terminated EC2 instances
// remain in the VPC, giving worker nodes time to finish shutting down before
// VPC resource cleanup runs.
func hfWaitVPCInstancesTerminated(ctx context.Context, ec2Client *ec2svc.Client, vpcID string, timeout time.Duration) {
	activeStates := []string{"pending", "running", "stopping", "stopped", "shutting-down"}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := ec2Client.DescribeInstances(ctx, &ec2svc.DescribeInstancesInput{
			Filters: []ec2types.Filter{
				{Name: awssdk.String("vpc-id"), Values: []string{vpcID}},
				{Name: awssdk.String("instance-state-name"), Values: activeStates},
			},
		})
		if err != nil {
			GinkgoWriter.Printf("DescribeInstances error while waiting for VPC %s to drain: %v\n", vpcID, err)
			return
		}
		count := 0
		for _, r := range out.Reservations {
			count += len(r.Instances)
		}
		if count == 0 {
			GinkgoWriter.Printf("All instances in VPC %s terminated\n", vpcID)
			return
		}
		GinkgoWriter.Printf("[%s] %d instance(s) still active in VPC %s, waiting...\n",
			time.Now().Format(time.RFC3339), count, vpcID)
		time.Sleep(15 * time.Second)
	}
	GinkgoWriter.Printf("Timed out waiting for instances in VPC %s to terminate; proceeding with cleanup\n", vpcID)
}
