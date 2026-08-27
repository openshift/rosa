package handler

import (
	"context"
	"crypto/sha1" //nolint:gosec // IAM OIDC thumbprints use SHA-1
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/tests/utils/helper"
	"github.com/openshift/rosa/tests/utils/log"
)

type platformAPIServiceAccount struct {
	namespace, name string
}

type platformAPIOperatorRole struct {
	suffix string
	policy string
	sas    []platformAPIServiceAccount
}

var platformAPIOperatorRoles = []platformAPIOperatorRole{
	{suffix: "-ingress", policy: "ROSAIngressOperatorPolicy", sas: []platformAPIServiceAccount{{"openshift-ingress-operator", "ingress-operator"}}},
	{suffix: "-cloud-controller-manager", policy: "ROSAKubeControllerPolicy", sas: []platformAPIServiceAccount{{"kube-system", "kube-controller-manager"}}},
	{suffix: "-ebs-csi", policy: "ROSAAmazonEBSCSIDriverOperatorPolicy", sas: []platformAPIServiceAccount{{"openshift-cluster-csi-drivers", "aws-ebs-csi-driver-operator"}, {"openshift-cluster-csi-drivers", "aws-ebs-csi-driver-controller-sa"}}},
	{suffix: "-image-registry", policy: "ROSAImageRegistryOperatorPolicy", sas: []platformAPIServiceAccount{{"openshift-image-registry", "cluster-image-registry-operator"}, {"openshift-image-registry", "registry"}}},
	{suffix: "-network-config", policy: "ROSACloudNetworkConfigOperatorPolicy", sas: []platformAPIServiceAccount{{"openshift-cloud-network-config-controller", "cloud-network-config-controller"}}},
	{suffix: "-control-plane-operator", policy: "ROSAControlPlaneOperatorPolicy", sas: []platformAPIServiceAccount{{"kube-system", "control-plane-operator"}}},
	{suffix: "-node-pool-management", policy: "ROSANodePoolManagementPolicy", sas: []platformAPIServiceAccount{{"kube-system", "capa-controller-manager"}}},
}

func (rh *resourcesHandler) PreparePlatformAPIPostCreateIAM(issuerURL, rolesPrefix string) error {
	if issuerURL == "" {
		return fmt.Errorf("OIDC issuer URL is required for Platform API post-create IAM")
	}
	awsClient, err := rh.GetAWSClient(false)
	if err != nil {
		return err
	}
	partition := platformAPIPartitionFromARN(awsClient.Arn)
	oidcProvider, err := platformAPIOIDCProvider(issuerURL)
	if err != nil {
		return err
	}
	thumbprint, err := platformAPIOIDCThumbprint(issuerURL)
	if err != nil {
		return err
	}

	ctx := context.Background()
	log.Logger.Info("Creating IAM OIDC provider for regional Platform API cluster")
	_, err = awsClient.IamClient.CreateOpenIDConnectProvider(ctx, &iam.CreateOpenIDConnectProviderInput{
		Url:            awssdk.String(issuerURL),
		ClientIDList:   []string{"openshift"},
		ThumbprintList: []string{thumbprint},
	})
	if err != nil && !isIAMEntityAlreadyExists(err) {
		return fmt.Errorf("creating OIDC provider: %w", err)
	}

	log.Logger.Infof("Creating operator IAM roles with prefix %s", rolesPrefix)
	for _, role := range platformAPIOperatorRoles {
		roleName := rolesPrefix + role.suffix
		trustPolicy := platformAPIOperatorTrustPolicy(partition, awsClient.AccountID, oidcProvider, role.sas)
		_, err = awsClient.IamClient.CreateRole(ctx, &iam.CreateRoleInput{
			RoleName:                 awssdk.String(roleName),
			AssumeRolePolicyDocument: awssdk.String(trustPolicy),
		})
		if err != nil && !isIAMEntityAlreadyExists(err) {
			return fmt.Errorf("creating role %s: %w", roleName, err)
		}
		policyARN := fmt.Sprintf("arn:%s:iam::aws:policy/service-role/%s", partition, role.policy)
		_, err = awsClient.IamClient.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
			RoleName:  awssdk.String(roleName),
			PolicyArn: awssdk.String(policyARN),
		})
		if err != nil {
			return fmt.Errorf("attaching policy to role %s: %w", roleName, err)
		}
	}

	workerRoleName := hyperfleet.ComputeInstanceProfile(rolesPrefix)
	log.Logger.Infof("Creating worker instance profile %s", workerRoleName)
	_, err = awsClient.IamClient.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName: awssdk.String(workerRoleName),
		AssumeRolePolicyDocument: awssdk.String(`{"Version":"2012-10-17","Statement":[{
			"Effect":"Allow","Principal":{"Service":"ec2.amazonaws.com"},"Action":"sts:AssumeRole"
		}]}`),
	})
	if err != nil && !isIAMEntityAlreadyExists(err) {
		return fmt.Errorf("creating worker role %s: %w", workerRoleName, err)
	}
	for _, policySuffix := range []string{
		"service-role/ROSAWorkerInstancePolicy",
		"AmazonSSMManagedInstanceCore",
	} {
		policyARN := fmt.Sprintf("arn:%s:iam::aws:policy/%s", partition, policySuffix)
		_, err = awsClient.IamClient.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
			RoleName:  awssdk.String(workerRoleName),
			PolicyArn: awssdk.String(policyARN),
		})
		if err != nil {
			return fmt.Errorf("attaching policy to worker role %s: %w", workerRoleName, err)
		}
	}
	_, err = awsClient.IamClient.CreateInstanceProfile(ctx, &iam.CreateInstanceProfileInput{
		InstanceProfileName: awssdk.String(workerRoleName),
	})
	if err != nil && !isIAMEntityAlreadyExists(err) {
		return fmt.Errorf("creating worker instance profile %s: %w", workerRoleName, err)
	}
	_, err = awsClient.IamClient.AddRoleToInstanceProfile(ctx, &iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: awssdk.String(workerRoleName),
		RoleName:            awssdk.String(workerRoleName),
	})
	if err != nil && !isIAMEntityAlreadyExists(err) {
		return fmt.Errorf("adding worker role to instance profile: %w", err)
	}
	return nil
}

func (rh *resourcesHandler) preparePlatformAPIHostedZone(hostedZoneName, vpcID string) (string, error) {
	awsClient, err := rh.GetAWSClient(false)
	if err != nil {
		return "", err
	}

	callerReference := helper.GenerateRandomString(10)
	hostedZoneOutput, err := awsClient.CreateHostedZone(
		hostedZoneName,
		callerReference,
		vpcID,
		rh.resources.Region,
		true,
	)
	if err != nil {
		return "", fmt.Errorf("creating hosted zone %s: %w", hostedZoneName, err)
	}

	hostedZoneID := strings.Split(*hostedZoneOutput.HostedZone.Id, "/")[2]
	if hostedZoneIsHCPInternal(hostedZoneName) {
		if regErr := rh.registerHostedCPInternalHostedZoneID(hostedZoneID); regErr != nil {
			log.Logger.Errorf("Error happened when record HCP Internal Hosted Zone ID: %s", regErr.Error())
		}
	} else if regErr := rh.registerIngressHostedZoneID(hostedZoneID); regErr != nil {
		log.Logger.Errorf("Error happened when record Ingress Hosted Zone ID: %s", regErr.Error())
	}

	log.Logger.Infof("Private hosted zone %s created: %s", hostedZoneName, hostedZoneID)
	return hostedZoneID, nil
}

func (rh *resourcesHandler) preparePlatformAPIWorkerSecurityGroup(clusterName string) error {
	if rh.vpc == nil {
		return fmt.Errorf("VPC required for worker security group")
	}
	awsClient, err := rh.GetAWSClient(false)
	if err != nil {
		return err
	}

	ctx := context.Background()
	sgName := clusterName + "-hc-worker-sg"
	sgOut, err := awsClient.Ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   awssdk.String(sgName),
		Description: awssdk.String("Worker node security group for " + clusterName),
		VpcId:       awssdk.String(rh.vpc.VpcID),
	})
	if err != nil {
		return fmt.Errorf("creating worker security group: %w", err)
	}
	sgID := awssdk.ToString(sgOut.GroupId)
	_, err = awsClient.Ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: awssdk.String(sgID),
		IpPermissions: []ec2types.IpPermission{
			{
				IpProtocol: awssdk.String("-1"),
				UserIdGroupPairs: []ec2types.UserIdGroupPair{
					{GroupId: awssdk.String(sgID)},
				},
			},
			{
				IpProtocol: awssdk.String("-1"),
				IpRanges: []ec2types.IpRange{
					{CidrIp: awssdk.String(rh.vpc.CIDRValue)},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("authorizing worker security group ingress: %w", err)
	}
	log.Logger.Infof("Worker security group %s created: %s", sgName, sgID)
	return nil
}

func isIAMEntityAlreadyExists(err error) bool {
	var alreadyExists *iamtypes.EntityAlreadyExistsException
	return err != nil && (errors.As(err, &alreadyExists) || strings.Contains(err.Error(), "EntityAlreadyExists"))
}

func platformAPIPartitionFromARN(arn string) string {
	parts := strings.SplitN(arn, ":", 3)
	if len(parts) >= 2 && parts[1] != "" {
		return parts[1]
	}
	return "aws"
}

func platformAPIOIDCProvider(issuerURL string) (string, error) {
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

func platformAPIOIDCThumbprint(issuerURL string) (string, error) {
	u, err := url.Parse(issuerURL)
	if err != nil {
		return "", fmt.Errorf("parsing issuer URL %q: %w", issuerURL, err)
	}
	port := u.Port()
	if port == "" {
		port = "443"
	}
	conn, err := tls.Dial("tcp", u.Hostname()+":"+port, &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec
	})
	if err != nil {
		return "", fmt.Errorf("TLS dial %s:%s: %w", u.Hostname(), port, err)
	}
	defer conn.Close()

	chain := conn.ConnectionState().PeerCertificates
	if len(chain) == 0 {
		return "", fmt.Errorf("no certificates in TLS chain for %s", u.Hostname())
	}
	sum := sha1.Sum(chain[len(chain)-1].Raw) //nolint:gosec
	return hex.EncodeToString(sum[:]), nil
}

func platformAPIOperatorTrustPolicy(partition, accountID, oidcProvider string, sas []platformAPIServiceAccount) string {
	subjects := make([]string, len(sas))
	for i, sa := range sas {
		subjects[i] = fmt.Sprintf("system:serviceaccount:%s:%s", sa.namespace, sa.name)
	}
	subjectValue := any(subjects[0])
	if len(subjects) > 1 {
		subjectValue = subjects
	}
	doc, _ := json.Marshal(map[string]any{
		"Version": "2012-10-17",
		"Statement": []map[string]any{{
			"Effect": "Allow",
			"Principal": map[string]string{
				"Federated": fmt.Sprintf("arn:%s:iam::%s:oidc-provider/%s", partition, accountID, oidcProvider),
			},
			"Action": "sts:AssumeRoleWithWebIdentity",
			"Condition": map[string]any{
				"StringEquals": map[string]any{oidcProvider + ":sub": subjectValue},
			},
		}},
	})
	return string(doc)
}
