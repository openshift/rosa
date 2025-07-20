package vpc_client

import (
	"fmt"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	CON "github.com/openshift-online/ocm-common/pkg/aws/consts"
	"github.com/openshift-online/ocm-common/pkg/file"
	"github.com/openshift-online/ocm-common/pkg/log"
)

// FindProxyLaunchImage will try to find a proper image based on the filters to launch the proxy instance
// No parameter needed here
// It will return an image ID and error if happens
func (vpc *VPC) FindProxyLaunchImage() (string, error) {
	filters := map[string][]string{
		"architecture": {
			"x86_64",
		},
		"state": {
			"available",
		},
		"image-type": {
			"machine",
		},
		"is-public": {
			"true",
		},
		"virtualization-type": {
			"hvm",
		},
		"root-device-type": {
			"ebs",
		},
	}

	output, err := vpc.AWSClient.DescribeImage([]string{}, filters)
	if err != nil {
		log.LogError("Describe image met error: %s", err)
		return "", err
	}
	if output == nil || len(output.Images) < 1 {
		log.LogError("Got the empty image via the filter: %s", filters)
		err = fmt.Errorf("got empty image list via the filter: %s", filters)
		return "", err
	}
	expectedImageID := ""
	nameRegexp := regexp.MustCompile(`al[0-9]{4}-ami[0-9\.-]*kernel[0-9-\._a-z]*`)
	for _, image := range output.Images {
		if nameRegexp.MatchString(*image.Name) {
			expectedImageID = *image.ImageId
			break
		}
	}
	if expectedImageID != "" {
		log.LogInfo("Got the image ID : %s", expectedImageID)
	} else {
		log.LogError("Got no proper image meet the regex: %s", nameRegexp.String())
		err = fmt.Errorf("got no proper image meet the regex: %s", nameRegexp.String())
	}

	return expectedImageID, err
}

// LaunchProxyInstance will launch a proxy instance on the indicated zone.
// If set imageID to empty, it will find the proxy image in the ProxyImageMap map
// LaunchProxyInstance will return proxyInstance detail, privateIPAddress,CAcontent and error
func (vpc *VPC) LaunchProxyInstance(zone string, keypairName string, privateKeyPath string) (inst types.Instance, privateIP string, proxyServerCA string, err error) {
	imageID, err := vpc.FindProxyLaunchImage()
	if err != nil {
		return inst, "", "", err
	}
	pubSubnet, err := vpc.PreparePublicSubnet(zone)
	if err != nil {
		log.LogInfo("Error preparing a subnet in current zone %s with image ID %s: %s", zone, imageID, err)
		return inst, "", "", err
	}
	SGID, err := vpc.CreateAndAuthorizeDefaultSecurityGroupForProxy()
	if err != nil {
		log.LogError("Prepare SG failed for the proxy preparation %s", err)
		return inst, "", "", err
	}

	keyName := fmt.Sprintf("%s-%s", CON.InstanceKeyNamePrefix, keypairName)
	key, err := vpc.CreateKeyPair(keyName)
	if err != nil {
		log.LogError("Create key pair %s failed %s", keyName, err)
		return inst, "", "", err
	}
	tags := map[string]string{
		"Name":  CON.ProxyName,
		"VpcId": vpc.VpcID,
	}
	_, err = vpc.AWSClient.TagResource(*key.KeyPairId, tags)
	if err != nil {
		log.LogError("Add tag for key pair %s failed %s", *key.KeyPairId, err)
		return inst, "", "", err
	}
	privateKeyName := fmt.Sprintf("%s-%s", keypairName, "keyPair.pem")
	sshKey, err := file.WriteToFile(*key.KeyMaterial, privateKeyName, privateKeyPath)
	if err != nil {
		log.LogError("Write private key to file failed %s", err)
		return inst, "", "", err
	}

	instOut, err := vpc.AWSClient.LaunchInstance(pubSubnet.ID, imageID, 1, "t3.medium", keyName, []string{SGID}, true)
	if err != nil {
		log.LogError("Launch proxy instance failed %s", err)
		return inst, "", "", err
	} else {
		log.LogInfo("Launch proxy instance %s succeed", *instOut.Instances[0].InstanceId)
	}

	instID := *instOut.Instances[0].InstanceId
	_, err = vpc.AWSClient.TagResource(instID, tags)
	if err != nil {
		log.LogError("Add tag for instance  %s failed %s", instID, err)
		return inst, "", "", err
	}

	publicIP, err := vpc.AWSClient.AllocateEIPAndAssociateInstance(instID)
	if err != nil {
		log.LogError("Prepare EIP failed for the proxy preparation %s", err)
		return inst, "", "", err
	}
	log.LogInfo("Prepare EIP successfully for the proxy preparation. Launch with IP: %s", publicIP)

	time.Sleep(2 * time.Minute)
	hostname := fmt.Sprintf("%s:22", publicIP)
	err = setupMITMProxyServer(sshKey, hostname)
	if err != nil {
		log.LogError("Setup MITM proxy server failed  %s", err)
		return inst, "", "", err
	}

	cmd := "cat mitm-ca.pem"
	caContent, err := Exec_CMD(CON.AWSInstanceUser, sshKey, hostname, cmd)
	if err != nil {
		log.LogError("login instance to run cmd %s:%s", cmd, err)
		return inst, "", "", err
	}
	return instOut.Instances[0], *instOut.Instances[0].PrivateIpAddress, caContent, err
}

func setupMITMProxyServer(sshKey string, hostname string) (err error) {
	setupProxyCMDs := []string{
		"sudo yum install -y wget",
		"wget https://snapshots.mitmproxy.org/7.0.2/mitmproxy-7.0.2-linux.tar.gz",
		"mkdir mitm",
		"tar zxvf mitmproxy-7.0.2-linux.tar.gz -C mitm",
		"nohup ./mitm/mitmdump --showhost --ssl-insecure > mitm.log 2>&1 &",
		"sleep 5",
		"http_proxy=127.0.0.1:8080 curl http://mitm.it/cert/pem -s > ~/mitm-ca.pem",
	}
	for _, cmd := range setupProxyCMDs {
		_, err = Exec_CMD(CON.AWSInstanceUser, sshKey, hostname, cmd)
		if err != nil {
			return err
		}
		log.LogDebug("Run the cmd successfully: %s", cmd)
	}
	return
}
