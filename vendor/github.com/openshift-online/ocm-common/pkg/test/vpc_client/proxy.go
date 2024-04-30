package vpc_client

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	CON "github.com/openshift-online/ocm-common/pkg/aws/consts"
	"github.com/openshift-online/ocm-common/pkg/log"
)

// LaunchProxyInstance will launch a proxy instance on the indicated zone.
// If set imageID to empty, it will find the proxy image in the ProxyImageMap map
// LaunchProxyInstance will return proxyInstance detail, privateIPAddress,CAcontent and error
func (vpc *VPC) LaunchProxyInstance(imageID string, zone string, sshKey string) (types.Instance, string, string, error) {
	var inst types.Instance
	if imageID == "" {
		var ok bool
		imageID, ok = CON.ProxyImageMap[vpc.Region]
		if !ok {
			log.LogInfo("Cannot find proxy image of region %s in map ProxyImageMap, will copy from existing region", vpc.Region)
			var err error
			imageID, err = vpc.CopyImageToProxy(CON.ProxyName)
			if err != nil {
				log.LogError("Error to copy image ID %s: %s", imageID, err)
				return inst, "", "", err
			}
			//Wait 30 minutes for image to active
			result, err := vpc.WaitImageToActive(imageID, 30)
			if err != nil || !result {
				log.LogError("Error wait image %s to active %s", imageID, err)
				return inst, "", "", err
			}
		}
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

	instOut, err := vpc.AWSClient.LaunchInstance(pubSubnet.ID, imageID, 1, "t3.medium", CON.InstanceKeyName, []string{SGID}, true)
	if err != nil {
		log.LogError("Launch proxy instance failed %s", err)
		return inst, "", "", err
	} else {
		log.LogInfo("Launch proxy instance %s succeed", *instOut.Instances[0].InstanceId)
	}
	tags := map[string]string{
		"Name": CON.ProxyName,
	}
	instID := *instOut.Instances[0].InstanceId
	_, err = vpc.AWSClient.TagResource(instID, tags)
	if err != nil {
		return inst, "", "", fmt.Errorf("tag instance %s failed:%s", instID, err)
	}

	publicIP, err := vpc.AWSClient.AllocateEIPAndAssociateInstance(instID)
	if err != nil {
		log.LogError("Prepare EIP failed for the proxy preparation %s", err)
		return inst, "", "", err
	}
	log.LogInfo("Prepare EIP successfully for the proxy preparation. Launch with IP: %s", publicIP)

	time.Sleep(2 * time.Minute)
	cmd1 := "http_proxy=127.0.0.1:8080 curl http://mitm.it/cert/pem -s > mitm-ca.pem"
	cmd2 := "cat mitm-ca.pem"
	hostname := fmt.Sprintf("%s:22", publicIP)
	_, err = Exec_CMD(CON.AWSInstanceUser, sshKey, hostname, cmd1)
	if err != nil {
		log.LogError("login instance to run cmd %s failed %s", cmd1, err)
		return inst, "", "", err
	}
	caContent, err := Exec_CMD(CON.AWSInstanceUser, sshKey, hostname, cmd2)
	if err != nil {
		log.LogError("login instance to run cmd %s failed %s", cmd2, err)
		return inst, "", "", err
	}

	return instOut.Instances[0], *instOut.Instances[0].PrivateIpAddress, caContent, err
}

func (vpc *VPC) CopyImageToProxy(name string) (destinationImageID string, err error) {
	sourceRegion := "us-west-2"
	sourceImageID, ok := CON.ProxyImageMap[sourceRegion]
	if !ok {
		log.LogError("Can't find image from region %s :%s", sourceRegion, err)
		return "", err
	}
	destinationImageID, err = vpc.AWSClient.CopyImage(sourceImageID, sourceRegion, name)
	if err != nil {
		log.LogError("Copy image %s meet error %s", sourceImageID, err)
		return "", err
	}
	return destinationImageID, nil
}

func (vpc *VPC) WaitImageToActive(imageID string, timeout time.Duration) (imageAvailable bool, err error) {
	log.LogInfo("Waiting for image %s status to active. Timeout after %v mins", imageID, timeout)
	startTime := time.Now()
	imageAvailable = false
	for time.Now().Before(startTime.Add(timeout * time.Minute)) {
		output, err := vpc.AWSClient.DescribeImage(imageID)
		if err != nil {
			log.LogError("Error happened when describe image status: %s", imageID)
			return imageAvailable, err
		}
		if string(output.Images[0].State) == "available" {
			imageAvailable = true
			return imageAvailable, nil
		}

		time.Sleep(time.Minute)
	}
	err = fmt.Errorf("timeout for waiting image active")
	return imageAvailable, err

}
