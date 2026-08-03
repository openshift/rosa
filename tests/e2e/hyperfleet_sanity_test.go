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
	stssvc "github.com/aws/aws-sdk-go-v2/service/sts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	rosacli "github.com/openshift/rosa/tests/utils/exec/rosacli"
	hfrest "github.com/openshift-online/rosa-hyperfleet-api/clientset/rest"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/wrappers"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/hyperfleet-operator/api/v1alpha1"

	"github.com/openshift/rosa/pkg/hyperfleet"
)

const (
	hfVPCReadyTimeout      = 2 * time.Minute
	hfSubnetReadyTimeout   = 2 * time.Minute
	hfClusterReadyInterval = 30 * time.Second
	hfClusterReadyTimeout  = 90 * time.Minute
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

			// ── Network setup ────────────────────────────────────────────────

			By("Creating VPC")
			vpcOut, err := ec2Client.CreateVpc(ctx, &ec2svc.CreateVpcInput{
				CidrBlock: awssdk.String("10.0.0.0/16"),
				TagSpecifications: []ec2types.TagSpecification{
					{
						ResourceType: ec2types.ResourceTypeVpc,
						Tags:         []ec2types.Tag{{Key: awssdk.String("Name"), Value: awssdk.String(clusterName + "-vpc")}},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred(), "creating VPC")
			vpcID := awssdk.ToString(vpcOut.Vpc.VpcId)
			DeferCleanup(func() {
				By("Cleanup: deleting VPC")
				_, _ = ec2Client.DeleteVpc(ctx, &ec2svc.DeleteVpcInput{VpcId: awssdk.String(vpcID)})
			})

			By("Waiting for VPC to become available")
			err = ec2svc.NewVpcAvailableWaiter(ec2Client).Wait(ctx,
				&ec2svc.DescribeVpcsInput{VpcIds: []string{vpcID}},
				hfVPCReadyTimeout,
			)
			Expect(err).NotTo(HaveOccurred(), "waiting for VPC %s to become available", vpcID)

			By("Creating subnet")
			az := region + "a"
			subnetOut, err := ec2Client.CreateSubnet(ctx, &ec2svc.CreateSubnetInput{
				VpcId:            awssdk.String(vpcID),
				CidrBlock:        awssdk.String("10.0.0.0/24"),
				AvailabilityZone: awssdk.String(az),
				TagSpecifications: []ec2types.TagSpecification{
					{
						ResourceType: ec2types.ResourceTypeSubnet,
						Tags:         []ec2types.Tag{{Key: awssdk.String("Name"), Value: awssdk.String(clusterName + "-subnet")}},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred(), "creating subnet")
			subnetID := awssdk.ToString(subnetOut.Subnet.SubnetId)
			DeferCleanup(func() {
				By("Cleanup: deleting subnet")
				_, _ = ec2Client.DeleteSubnet(ctx, &ec2svc.DeleteSubnetInput{SubnetId: awssdk.String(subnetID)})
			})

			By("Waiting for subnet to become available")
			err = ec2svc.NewSubnetAvailableWaiter(ec2Client).Wait(ctx,
				&ec2svc.DescribeSubnetsInput{SubnetIds: []string{subnetID}},
				hfSubnetReadyTimeout,
			)
			Expect(err).NotTo(HaveOccurred(), "waiting for subnet %s to become available", subnetID)

			// ── Cluster create ───────────────────────────────────────────────

			By("Creating cluster via CLI")
			// HYPERFLEET_VERSION is optional — when empty the server resolves a default.
			version := os.Getenv("HYPERFLEET_VERSION")
			createArgs := []string{
				"--hyperfleet-url", hfURL,
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

			DeferCleanup(func() {
				By("Cleanup: deleting cluster via CLI")
				_, _ = rosacli.NewClient().Runner.
					Cmd("delete", "cluster").
					CmdFlags("--hyperfleet-url", hfURL, "-c", clusterName, "-y").
					Run()
			})

			By("Fetching cluster ID and OIDC IssuerURL via CLI describe")
			describeCreateRunner := rosacli.NewClient().Runner
			describeCreateRunner.JsonFormat()
			describeCreateOut, err := describeCreateRunner.Cmd("describe", "cluster").
				CmdFlags("--hyperfleet-url", hfURL, "-c", clusterName).
				Run()
			Expect(err).NotTo(HaveOccurred(), "rosa describe cluster after create")

			var createDescribeMap map[string]interface{}
			Expect(json.Unmarshal(describeCreateOut.Bytes(), &createDescribeMap)).To(Succeed(),
				"parsing describe JSON output after create")
			clusterID, _ := createDescribeMap["id"].(string)
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
			oidcARN := awssdk.ToString(oidcOut.OpenIDConnectProviderArn)
			GinkgoWriter.Printf("OIDC provider ARN: %s\n", oidcARN)

			DeferCleanup(func() {
				By("Cleanup: deleting OIDC provider")
				_, _ = iamClient.DeleteOpenIDConnectProvider(ctx, &iamsvc.DeleteOpenIDConnectProviderInput{
					OpenIDConnectProviderArn: awssdk.String(oidcARN),
				})
			})

			// ── Operator IAM roles ───────────────────────────────────────────

			By("Creating operator IAM roles with OIDC trust policies")
			createdRoles := make([]string, 0, len(hfOperatorRoles))

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

			DeferCleanup(func() {
				By("Cleanup: detaching policies and deleting operator roles")
				for i, role := range hfOperatorRoles {
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
			listOut, err := rosaRunner.Cmd("list", "clusters").CmdFlags("--hyperfleet-url", hfURL).Run()
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
				CmdFlags("--hyperfleet-url", hfURL, "-c", clusterName).
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
		})
	},
)

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
