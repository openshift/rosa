package vpc_client

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	CON "github.com/openshift-online/ocm-common/pkg/aws/consts"
	"github.com/openshift-online/ocm-common/pkg/log"
)

// LaunchBastion will launch a bastion instance on the indicated zone.
// If set imageID to empty, it will find the bastion image in the bastionImageMap map
func (vpc *VPC) LaunchBastion(imageID string, zone string) (*types.Instance, error) {
	var inst *types.Instance
	if imageID == "" {
		var ok bool
		imageID, ok = CON.BastionImageMap[vpc.Region]
		if !ok {
			log.LogError("Cannot find bastion image of region %s in map bastionImageMap, please indicate it as parameter", vpc.Region)
			return nil, fmt.Errorf("cannot find bastion image of region %s in map bastionImageMap, please indicate it as parameter", vpc.Region)
		}
	}
	pubSubnet, err := vpc.PreparePublicSubnet(zone)
	if err != nil {
		log.LogInfo("Error preparing a subnet in current zone %s with image ID %s: %s", zone, imageID, err)
		return nil, err
	}
	SGID, err := vpc.CreateAndAuthorizeDefaultSecurityGroupForProxy()
	if err != nil {
		log.LogError("Prepare SG failed for the bastion preparation %s", err)
		return inst, err
	}

	key, err := vpc.CreateKeyPair(fmt.Sprintf("%s-bastion", CON.InstanceKeyNamePrefix))
	if err != nil {
		log.LogError("Create key pair failed %s", err)
		return inst, err
	}
	instOut, err := vpc.AWSClient.LaunchInstance(pubSubnet.ID, imageID, 1, "t3.medium", *key.KeyName, []string{SGID}, true)

	if err != nil {
		log.LogError("Launch bastion instance failed %s", err)
		return inst, err
	} else {
		log.LogInfo("Launch bastion instance %s succeed", *instOut.Instances[0].InstanceId)
	}
	tags := map[string]string{
		"Name": CON.BastionName,
	}
	instID := *instOut.Instances[0].InstanceId
	_, err = vpc.AWSClient.TagResource(instID, tags)
	if err != nil {
		return inst, fmt.Errorf("tag instance %s failed:%s", instID, err)
	}

	publicIP, err := vpc.AWSClient.AllocateEIPAndAssociateInstance(instID)
	if err != nil {
		log.LogError("Prepare EIP failed for the bastion preparation %s", err)
		return inst, err
	}
	log.LogInfo("Prepare EIP successfully for the bastion preparation. Launch with IP: %s", publicIP)
	inst = &instOut.Instances[0]
	inst.PublicIpAddress = &publicIP
	time.Sleep(2 * time.Minute)
	return inst, nil
}

func (vpc *VPC) PrepareBastion(zone string) (*types.Instance, error) {
	filters := []map[string][]string{
		{
			"vpc-id": {
				vpc.VpcID,
			},
		},
		{
			"tag:Name": {
				CON.BastionName,
			},
		},
	}

	insts, err := vpc.AWSClient.ListInstances([]string{}, filters...)
	if err != nil {
		return nil, err
	}
	if len(insts) == 0 {
		log.LogInfo("Didn't found an existing bastion, going to launch one")
		return vpc.LaunchBastion("", zone)

	}
	log.LogInfo("Found existing bastion: %s", *insts[0].InstanceId)
	return &insts[0], nil
}
