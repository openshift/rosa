package e2e

import (
	"context"
	"crypto/sha1" //nolint:gosec
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	ec2svc "github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	iamsvc "github.com/aws/aws-sdk-go-v2/service/iam"
	route53svc "github.com/aws/aws-sdk-go-v2/service/route53"
	route53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	stssvc "github.com/aws/aws-sdk-go-v2/service/sts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	hfrest "github.com/openshift-online/rosa-hyperfleet-api/clientset/rest"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/wrappers"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/hyperfleet-operator/api/v1alpha1"

	"github.com/openshift/rosa/pkg/hyperfleet"
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
)

// hfOperatorRole maps a role suffix to its AWS managed policy name and the
// service accounts that should be allowed to assume it.
type hfOperatorRole struct {
	suffix          string
	policy          string
	serviceAccounts []hfServiceAccount
}

type hfServiceAccount struct {
	namespace string
	name      string
}

var hfOperatorRoles = []hfOperatorRole{
	{
		suffix: "-ingress",
		policy: "ROSAIngressOperatorPolicy",
		serviceAccounts: []hfServiceAccount{
			{namespace: "openshift-ingress-operator", name: "ingress-operator"},
		},
	},
	{
		suffix: "-cloud-controller-manager",
		policy: "ROSAKubeControllerPolicy",
		serviceAccounts: []hfServiceAccount{
			{namespace: "kube-system", name: "kube-controller-manager"},
		},
	},
	{
		suffix: "-ebs-csi",
		policy: "ROSAAmazonEBSCSIDriverOperatorPolicy",
		serviceAccounts: []hfServiceAccount{
			{namespace: "openshift-cluster-csi-drivers", name: "aws-ebs-csi-driver-operator"},
			{namespace: "openshift-cluster-csi-drivers", name: "aws-ebs-csi-driver-controller-sa"},
		},
	},
	{
		suffix: "-image-registry",
		policy: "ROSAImageRegistryOperatorPolicy",
		serviceAccounts: []hfServiceAccount{
			{namespace: "openshift-image-registry", name: "cluster-image-registry-operator"},
			{namespace: "openshift-image-registry", name: "registry"},
		},
	},
	{
		suffix: "-network-config",
		policy: "ROSACloudNetworkConfigOperatorPolicy",
		serviceAccounts: []hfServiceAccount{
			{namespace: "openshift-cloud-network-config-controller", name: "cloud-network-config-controller"},
		},
	},
	{
		suffix: "-control-plane-operator",
		policy: "ROSAControlPlaneOperatorPolicy",
		serviceAccounts: []hfServiceAccount{
			{namespace: "kube-system", name: "control-plane-operator"},
		},
	},
	{
		suffix: "-node-pool-management",
		policy: "ROSANodePoolManagementPolicy",
		serviceAccounts: []hfServiceAccount{
			{namespace: "kube-system", name: "capa-controller-manager"},
		},
	},
}

var _ = Describe("Hyperfleet sanity",
	func() {
		It("creates and deletes an HCP cluster via the Platform API", func() {
			hfURL := os.Getenv("HYPERFLEET_URL")
			if hfURL == "" {
				Skip("HYPERFLEET_URL is not set")
			}
			clusterName := os.Getenv("CLUSTER_NAME")
			if clusterName == "" {
				clusterName = fmt.Sprintf("hf-sanity-%d", time.Now().Unix())
			}
			rolesPrefix := os.Getenv("OPERATOR_ROLES_PREFIX")
			if rolesPrefix == "" {
				rolesPrefix = clusterName
			}

			By("Logging in with Platform API URL")
			_, err := rosacli.NewClient().Runner.
				Cmd("login").
				CmdFlags("--hyperfleet-url", hfURL).
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa login --hyperfleet-url")

			DeferCleanup(func() {
				By("Cleanup: removing CLI config")
				_, _ = rosacli.NewClient().Runner.Cmd("logout").Run()
			})

			By("Verifying whoami shows Platform API URL")
			whoamiRunner := rosacli.NewClient().Runner
			whoamiRunner.JsonFormat()
			whoamiOut, err := whoamiRunner.Cmd("whoami").Run()
			Expect(err).NotTo(HaveOccurred(), "rosa whoami")
			var whoamiMap map[string]interface{}
			Expect(json.Unmarshal(whoamiOut.Bytes(), &whoamiMap)).To(Succeed(),
				"parsing whoami JSON output")
			Expect(whoamiMap["Platform API"]).To(Equal(hfURL),
				"whoami must report the Platform API URL stored during login")

			ctx := context.Background()

			// Derive region from URL or AWS_DEFAULT_REGION.
			region, err := hyperfleet.ExtractRegion(hfURL)
			if err != nil {
				envRegion := os.Getenv("AWS_DEFAULT_REGION")
				Expect(envRegion).NotTo(BeEmpty(),
					"cannot derive region from HYPERFLEET_URL; set AWS_DEFAULT_REGION")
				region = envRegion
			}

			By("Loading AWS configuration")
			awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
			Expect(err).NotTo(HaveOccurred(), "loading AWS config")

			By("Resolving AWS caller identity")
			stsClient := stssvc.NewFromConfig(awsCfg)
			identity, err := stsClient.GetCallerIdentity(ctx, &stssvc.GetCallerIdentityInput{})
			Expect(err).NotTo(HaveOccurred(), "STS GetCallerIdentity")
			accountID := awssdk.ToString(identity.Account)
			callerARN := awssdk.ToString(identity.Arn)
			partition := hfPartitionFromARN(callerARN)

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
			iamClient := iamsvc.NewFromConfig(awsCfg)
			r53Client := route53svc.NewFromConfig(awsCfg)

			// ── Network setup ────────────────────────────────────────────────
			// Mirrors what rosactl cluster-vpc create provisions via CloudFormation.

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
			DeferCleanup(func() {
				By("Cleanup: deleting VPC")
				_, _ = ec2Client.DeleteVpc(ctx, &ec2svc.DeleteVpcInput{VpcId: awssdk.String(vpcID)})
			})

			By("Waiting for VPC to become available")
			Expect(ec2svc.NewVpcAvailableWaiter(ec2Client).Wait(ctx,
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
			DeferCleanup(func() {
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
			DeferCleanup(func() {
				By("Cleanup: deleting public subnet")
				_, _ = ec2Client.DeleteSubnet(ctx, &ec2svc.DeleteSubnetInput{SubnetId: awssdk.String(publicSubnetID)})
			})

			By("Waiting for subnets to become available")
			Expect(ec2svc.NewSubnetAvailableWaiter(ec2Client).Wait(ctx,
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
			_, err = ec2Client.AttachInternetGateway(ctx, &ec2svc.AttachInternetGatewayInput{
				InternetGatewayId: awssdk.String(igwID),
				VpcId:             awssdk.String(vpcID),
			})
			Expect(err).NotTo(HaveOccurred(), "attaching IGW %s to VPC %s", igwID, vpcID)
			DeferCleanup(func() {
				By("Cleanup: detaching and deleting Internet Gateway")
				_, _ = ec2Client.DetachInternetGateway(ctx, &ec2svc.DetachInternetGatewayInput{
					InternetGatewayId: awssdk.String(igwID),
					VpcId:             awssdk.String(vpcID),
				})
				_, _ = ec2Client.DeleteInternetGateway(ctx, &ec2svc.DeleteInternetGatewayInput{
					InternetGatewayId: awssdk.String(igwID),
				})
			})

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
			DeferCleanup(func() {
				By("Cleanup: deleting NAT Gateway and releasing EIP")
				_, _ = ec2Client.DeleteNatGateway(ctx, &ec2svc.DeleteNatGatewayInput{
					NatGatewayId: awssdk.String(natGWID),
				})
				_ = ec2svc.NewNatGatewayDeletedWaiter(ec2Client).Wait(ctx,
					&ec2svc.DescribeNatGatewaysInput{NatGatewayIds: []string{natGWID}},
					5*time.Minute,
				)
				_, _ = ec2Client.ReleaseAddress(ctx, &ec2svc.ReleaseAddressInput{
					AllocationId: awssdk.String(natEIPAllocID),
				})
			})

			By("Waiting for NAT Gateway to become available")
			Expect(ec2svc.NewNatGatewayAvailableWaiter(ec2Client).Wait(ctx,
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
			publicRTAssocID := awssdk.ToString(pubAssocOut.AssociationId)

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
			privateRTAssocID := awssdk.ToString(privAssocOut.AssociationId)
			DeferCleanup(func() {
				By("Cleanup: deleting route tables")
				_, _ = ec2Client.DisassociateRouteTable(ctx, &ec2svc.DisassociateRouteTableInput{
					AssociationId: awssdk.String(privateRTAssocID),
				})
				_, _ = ec2Client.DisassociateRouteTable(ctx, &ec2svc.DisassociateRouteTableInput{
					AssociationId: awssdk.String(publicRTAssocID),
				})
				_, _ = ec2Client.DeleteRouteTable(ctx, &ec2svc.DeleteRouteTableInput{
					RouteTableId: awssdk.String(privateRTID),
				})
				_, _ = ec2Client.DeleteRouteTable(ctx, &ec2svc.DeleteRouteTableInput{
					RouteTableId: awssdk.String(publicRTID),
				})
			})

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
			DeferCleanup(func() {
				By("Cleanup: deleting worker security group")
				_, _ = ec2Client.DeleteSecurityGroup(ctx, &ec2svc.DeleteSecurityGroupInput{
					GroupId: awssdk.String(workerSGID),
				})
			})

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
			DeferCleanup(func() {
				By("Cleanup: deleting private hosted zone")
				_, _ = r53Client.DeleteHostedZone(ctx, &route53svc.DeleteHostedZoneInput{
					Id: awssdk.String(hostedZoneID),
				})
			})

			// ── IAM cleanup (pre-registered to execute after cluster deletion) ─────
			// DeferCleanup runs in LIFO order. Registering these before the cluster
			// delete DeferCleanup ensures IAM resources are removed only after the
			// cluster and its worker nodes are fully gone. The OIDC provider must
			// remain available while the cluster is deleting so operators can
			// authenticate to AWS; it is deleted last among these.

			var oidcARN string
			DeferCleanup(func() {
				if oidcARN == "" {
					return
				}
				By("Cleanup: deleting OIDC provider")
				_, _ = iamClient.DeleteOpenIDConnectProvider(ctx, &iamsvc.DeleteOpenIDConnectProviderInput{
					OpenIDConnectProviderArn: awssdk.String(oidcARN),
				})
			})

			var createdRoles []string
			DeferCleanup(func() {
				By("Cleanup: detaching policies and deleting operator roles")
				for i, role := range hfOperatorRoles {
					if i >= len(createdRoles) {
						break
					}
					roleName := createdRoles[i]
					policyARN := fmt.Sprintf("arn:%s:iam::aws:policy/service-role/%s", partition, role.policy)
					_, _ = iamClient.DetachRolePolicy(ctx, &iamsvc.DetachRolePolicyInput{
						RoleName:  awssdk.String(roleName),
						PolicyArn: awssdk.String(policyARN),
					})
					_, _ = iamClient.DeleteRole(ctx, &iamsvc.DeleteRoleInput{
						RoleName: awssdk.String(roleName),
					})
				}
			})

			workerRoleName := hyperfleet.ComputeInstanceProfile(rolesPrefix)
			workerPolicies := []string{
				fmt.Sprintf("arn:%s:iam::aws:policy/service-role/ROSAWorkerInstancePolicy", partition),
				fmt.Sprintf("arn:%s:iam::aws:policy/AmazonSSMManagedInstanceCore", partition),
			}
			DeferCleanup(func() {
				By("Cleanup: deleting worker instance profile and role")
				_, _ = iamClient.RemoveRoleFromInstanceProfile(ctx, &iamsvc.RemoveRoleFromInstanceProfileInput{
					InstanceProfileName: awssdk.String(workerRoleName),
					RoleName:            awssdk.String(workerRoleName),
				})
				_, _ = iamClient.DeleteInstanceProfile(ctx, &iamsvc.DeleteInstanceProfileInput{
					InstanceProfileName: awssdk.String(workerRoleName),
				})
				for _, policyARN := range workerPolicies {
					_, _ = iamClient.DetachRolePolicy(ctx, &iamsvc.DetachRolePolicyInput{
						RoleName:  awssdk.String(workerRoleName),
						PolicyArn: awssdk.String(policyARN),
					})
				}
				_, _ = iamClient.DeleteRole(ctx, &iamsvc.DeleteRoleInput{
					RoleName: awssdk.String(workerRoleName),
				})
			})

			// ── Cluster create ───────────────────────────────────────────────

			By("Creating cluster via CLI")
			// HYPERFLEET_VERSION is optional — when empty the server resolves a default.
			version := os.Getenv("HYPERFLEET_VERSION")
			createArgs := []string{
				"--cluster-name", clusterName,
				"--subnet-ids", subnetID,
				"--operator-roles-prefix", rolesPrefix,
			}
			if version != "" {
				createArgs = append(createArgs, "--version", version)
			}
			_, err = rosacli.NewClient().Runner.
				Cmd("create", "cluster").
				CmdFlags(createArgs...).
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa create cluster CLI call")

			// Declare clusterID before the DeferCleanup so the closure captures
			// the variable; it will be populated after the describe call below.
			var clusterID string
			DeferCleanup(func() {
				By("Cleanup: deleting cluster via CLI")
				_, _ = rosacli.NewClient().Runner.
					Cmd("delete", "cluster").
					CmdFlags("-c", clusterName, "-y").
					Run()

				if clusterID == "" {
					return
				}
				By("Waiting for cluster to be fully deleted before releasing AWS resources")
				_ = hfClient.HyperfleetV1alpha1().Clusters(accountID).WaitUntil(
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

				By("Releasing orphaned ENIs left by cluster controllers")
				hfDeleteAvailableENIs(ctx, ec2Client, vpcID)
			})

			By("Fetching cluster ID and OIDC IssuerURL via CLI describe")
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
			issuerURL, _ := specMap["oidc_issuer"].(string)
			Expect(issuerURL).NotTo(BeEmpty(), "OIDC IssuerURL must be present in describe response after create")
			GinkgoWriter.Printf("OIDC IssuerURL: %s\n", issuerURL)

			// oidcProvider is the host+path without the https:// scheme prefix,
			// used as the principal in trust policies and in OIDC condition keys.
			oidcProvider, err := hfOIDCProvider(issuerURL)
			Expect(err).NotTo(HaveOccurred(), "parsing OIDC provider from IssuerURL")

			// ── OIDC provider ────────────────────────────────────────────────

			By("Fetching OIDC thumbprint")
			thumbprint, err := hfOIDCThumbprint(issuerURL)
			Expect(err).NotTo(HaveOccurred(), "computing OIDC thumbprint")

			By("Creating OIDC provider in IAM")
			oidcOut, err := iamClient.CreateOpenIDConnectProvider(ctx, &iamsvc.CreateOpenIDConnectProviderInput{
				Url:            awssdk.String(issuerURL),
				ClientIDList:   []string{"openshift"},
				ThumbprintList: []string{thumbprint},
			})
			Expect(err).NotTo(HaveOccurred(), "creating OIDC provider")
			oidcARN = awssdk.ToString(oidcOut.OpenIDConnectProviderArn)
			GinkgoWriter.Printf("OIDC provider ARN: %s\n", oidcARN)

			// ── Operator IAM roles ───────────────────────────────────────────

			By("Creating operator IAM roles with OIDC trust policies")
			for _, role := range hfOperatorRoles {
				roleName := rolesPrefix + role.suffix
				trustPolicy := hfBuildTrustPolicy(partition, accountID, oidcProvider, role.serviceAccounts)

				_, err := iamClient.CreateRole(ctx, &iamsvc.CreateRoleInput{
					RoleName:                 awssdk.String(roleName),
					AssumeRolePolicyDocument: awssdk.String(trustPolicy),
					Description:              awssdk.String("ROSA HCP operator role (hyperfleet sanity test)"),
				})
				Expect(err).NotTo(HaveOccurred(), "creating role %s", roleName)
				createdRoles = append(createdRoles, roleName)
			}

			By("Attaching managed policies to operator roles")
			for i, role := range hfOperatorRoles {
				roleName := createdRoles[i]
				policyARN := fmt.Sprintf("arn:%s:iam::aws:policy/service-role/%s", partition, role.policy)
				_, err := iamClient.AttachRolePolicy(ctx, &iamsvc.AttachRolePolicyInput{
					RoleName:  awssdk.String(roleName),
					PolicyArn: awssdk.String(policyARN),
				})
				Expect(err).NotTo(HaveOccurred(), "attaching policy %s to role %s", policyARN, roleName)
			}

			// ── Worker IAM role + instance profile ───────────────────────────

			By("Creating worker IAM role for node instances")
			_, err = iamClient.CreateRole(ctx, &iamsvc.CreateRoleInput{
				RoleName: awssdk.String(workerRoleName),
				AssumeRolePolicyDocument: awssdk.String(`{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {"Service": "ec2.amazonaws.com"},
    "Action": "sts:AssumeRole"
  }]
}`),
				Description: awssdk.String("ROSA HCP worker node role (hyperfleet sanity test)"),
			})
			Expect(err).NotTo(HaveOccurred(), "creating worker role %s", workerRoleName)

			for _, policyARN := range workerPolicies {
				_, err = iamClient.AttachRolePolicy(ctx, &iamsvc.AttachRolePolicyInput{
					RoleName:  awssdk.String(workerRoleName),
					PolicyArn: awssdk.String(policyARN),
				})
				Expect(err).NotTo(HaveOccurred(), "attaching %s to worker role", policyARN)
			}

			By("Creating worker IAM instance profile")
			_, err = iamClient.CreateInstanceProfile(ctx, &iamsvc.CreateInstanceProfileInput{
				InstanceProfileName: awssdk.String(workerRoleName),
			})
			Expect(err).NotTo(HaveOccurred(), "creating instance profile %s", workerRoleName)

			_, err = iamClient.AddRoleToInstanceProfile(ctx, &iamsvc.AddRoleToInstanceProfileInput{
				InstanceProfileName: awssdk.String(workerRoleName),
				RoleName:            awssdk.String(workerRoleName),
			})
			Expect(err).NotTo(HaveOccurred(), "adding worker role to instance profile")

			// ── Assertions ───────────────────────────────────────────────────

			By("Waiting for cluster to become Ready")
			var clusterPhase v1alpha1.ClusterPhase
			err = hfClient.HyperfleetV1alpha1().Clusters(accountID).WaitUntil(
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

			// ── Describe sanity ───────────────────────────────────────────────

			By("Describing cluster via CLI and comparing with Get response")
			getOut, err := hfClient.HyperfleetV1alpha1().Clusters(accountID).Get(
				ctx, clusterID, wrappers.GetOptions{},
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
					getOut.Status.ControlPlaneEndpoint.Port)),
				"CLI describe api_url must match Get control plane endpoint")

			GinkgoWriter.Printf("rosa describe cluster output:\n%s\n", cliOut.String())

			// ── Node pool lifecycle ───────────────────────────────────────────

			instanceType := os.Getenv("HYPERFLEET_INSTANCE_TYPE")
			if instanceType == "" {
				instanceType = hfDefaultInstanceType
			}
			const np1Name = "np1"
			const np2Name = "np2"
			var np1ID, np2ID string

			DeferCleanup(func() {
				// np2 is explicitly deleted and waited on in the happy path.
				// np1 cannot be waited on due to PDB restrictions — cluster
				// deletion forces it; fire-and-forget here as a safety net.
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
			npList, err := hfClient.HyperfleetV1alpha1().NodePools(clusterID).List(ctx, wrappers.ListOptions{})
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
					GinkgoWriter.Printf("[%s] node pool %s: phase=%s\n",
						time.Now().Format(time.RFC3339), np1Name, n.Status.Phase)
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
					GinkgoWriter.Printf("[%s] node pool %s: phase=%s\n",
						time.Now().Format(time.RFC3339), np2Name, n.Status.Phase)
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

			By("Deleting node pool 2 via CLI")
			_, err = rosacli.NewClient().Runner.
				Cmd("delete", "machinepool").
				CmdFlags("-c", clusterName, "--machinepool", np2Name, "--yes").
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa delete machinepool %s", np2Name)

			By("Waiting for node pool 2 to be deleted")
			Expect(hfClient.HyperfleetV1alpha1().NodePools(clusterID).WaitUntil(ctx, np2ID,
				func(n *v1alpha1.NodePool) bool {
					if n == nil {
						GinkgoWriter.Printf("NodePool %s deleted\n", np2Name)
						return true
					}
					GinkgoWriter.Printf("[%s] node pool %s: phase=%s, waiting for deletion\n",
						time.Now().Format(time.RFC3339), np2Name, n.Status.Phase)
					return false
				},
				hfNodePoolReadyInterval, hfNodePoolReadyTimeout,
			)).To(Succeed(), "waiting for node pool %s to be deleted", np2Name)

			// np1 (2 replicas) is the last node pool; default PDB prevents
			// draining its nodes so deletion cannot complete until the cluster
			// itself is deleted. Initiate the request so the operator begins
			// cleanup, then let the cluster delete drive the final teardown.
			By("Initiating node pool 1 deletion (cluster delete will complete it)")
			_, err = rosacli.NewClient().Runner.
				Cmd("delete", "machinepool").
				CmdFlags("-c", clusterName, "--machinepool", np1Name, "--yes").
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa delete machinepool %s", np1Name)
		})
	},
)

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

// hfPartitionFromARN returns the AWS partition extracted from an ARN string,
// defaulting to "aws" when the ARN cannot be parsed.
func hfPartitionFromARN(callerARN string) string {
	parts := strings.SplitN(callerARN, ":", 5)
	if len(parts) >= 2 && parts[1] != "" {
		return parts[1]
	}
	return "aws"
}

// hfOIDCProvider strips the https:// scheme from an issuer URL and returns
// the host+path string used as the IAM OIDC provider identifier.
func hfOIDCProvider(issuerURL string) (string, error) {
	u, err := url.Parse(issuerURL)
	if err != nil {
		return "", fmt.Errorf("parsing issuer URL %q: %w", issuerURL, err)
	}
	provider := u.Host
	if p := strings.TrimPrefix(u.Path, "/"); p != "" {
		provider = provider + "/" + p
	}
	return provider, nil
}

// hfOIDCThumbprint connects to the OIDC issuer host via TLS and returns the
// hex-encoded SHA-1 fingerprint of the root CA certificate in the chain.
// This is the value that IAM CreateOpenIDConnectProvider expects in ThumbprintList.
func hfOIDCThumbprint(issuerURL string) (string, error) {
	u, err := url.Parse(issuerURL)
	if err != nil {
		return "", fmt.Errorf("parsing issuer URL %q: %w", issuerURL, err)
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "443"
	}
	conn, err := tls.Dial("tcp", host+":"+port, &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // thumbprint derivation requires seeing the raw cert chain
	})
	if err != nil {
		return "", fmt.Errorf("TLS dial %s:%s: %w", host, port, err)
	}
	defer conn.Close()

	chain := conn.ConnectionState().PeerCertificates
	if len(chain) == 0 {
		return "", fmt.Errorf("no certificates in TLS chain for %s", host)
	}
	root := chain[len(chain)-1]
	sum := sha1.Sum(root.Raw) //nolint:gosec
	return hex.EncodeToString(sum[:]), nil
}

// hfBuildTrustPolicy returns a JSON assume-role policy document that allows
// the given OIDC provider to mint tokens for each service account listed.
func hfBuildTrustPolicy(partition, accountID, oidcProvider string, sas []hfServiceAccount) string {
	subjects := make([]string, 0, len(sas))
	for _, sa := range sas {
		subjects = append(subjects, fmt.Sprintf("system:serviceaccount:%s:%s", sa.namespace, sa.name))
	}

	type conditionValue interface{}
	var subjectValue conditionValue
	if len(subjects) == 1 {
		subjectValue = subjects[0]
	} else {
		subjectValue = subjects
	}

	doc := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect": "Allow",
				"Principal": map[string]string{
					"Federated": fmt.Sprintf("arn:%s:iam::%s:oidc-provider/%s", partition, accountID, oidcProvider),
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": map[string]interface{}{
					"StringEquals": map[string]interface{}{
						oidcProvider + ":sub": subjectValue,
					},
				},
			},
		},
	}
	b, _ := json.Marshal(doc)
	return string(b)
}
