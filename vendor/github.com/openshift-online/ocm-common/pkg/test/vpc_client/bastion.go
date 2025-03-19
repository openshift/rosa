package vpc_client

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/openshift-online/ocm-common/pkg/file"
	"golang.org/x/crypto/bcrypt"
	"net/url"
	"strings"
	"time"

	CON "github.com/openshift-online/ocm-common/pkg/aws/consts"
	awsUtils "github.com/openshift-online/ocm-common/pkg/aws/utils"
	"github.com/openshift-online/ocm-common/pkg/log"
	"github.com/openshift-online/ocm-common/pkg/utils"
)

// LaunchBastion will launch a bastion instance on the indicated zone.
// If set imageID to empty, it will find the bastion image using filter with specific name.
func (vpc *VPC) LaunchBastion(imageID string, zone string, userData string, keypairName string,
	privateKeyPath string) (*types.Instance, error) {
	var inst *types.Instance
	if imageID == "" {

		var err error
		imageID, err = vpc.FindProxyLaunchImage()
		if err != nil {
			log.LogError("Cannot find bastion image of region %s in map bastionImageMap, please indicate it as parameter", vpc.Region)
			return nil, err
		}
	}
	if userData == "" {
		log.LogError("Userdata can not be empty, pleas provide the correct userdata")
		return nil, errors.New("userData should not be empty")
	}
	pubSubnet, err := vpc.PreparePublicSubnet(zone)
	if err != nil {
		log.LogError("Error preparing a subnet in current zone %s with image ID %s: %s", zone, imageID, err)
		return nil, err
	}
	SGID, err := vpc.CreateAndAuthorizeDefaultSecurityGroupForProxy(3128)
	if err != nil {
		log.LogError("Prepare SG failed for the bastion preparation %s", err)
		return inst, err
	}
	keyName := fmt.Sprintf("%s-%s", CON.InstanceKeyNamePrefix, keypairName)
	key, err := vpc.CreateKeyPair(keyName)
	if err != nil {
		log.LogError("Create key pair failed %s", err)
		return inst, err
	}
	tags := map[string]string{
		"Name": CON.BastionName,
	}
	_, err = vpc.AWSClient.TagResource(*key.KeyPairId, tags)
	if err != nil {
		log.LogError("Add tag for key pair %s failed %s", *key.KeyPairId, err)
		return inst, err
	}

	privateKeyName := fmt.Sprintf("%s-%s", keypairName, "keyPair.pem")
	sshKeyPath, err := file.WriteToFile(*key.KeyMaterial, privateKeyName, privateKeyPath)
	if err != nil {
		log.LogError("Write private key to %s failed %s", sshKeyPath, err)
		return inst, err
	}
	instOut, err := vpc.AWSClient.LaunchInstance(pubSubnet.ID, imageID, 1, "t3.medium", *key.KeyName,
		[]string{SGID}, true, userData)

	if err != nil {
		log.LogError("Launch bastion instance failed %s", err)
		return inst, err
	} else {
		log.LogInfo("Launch bastion instance %s succeed", *instOut.Instances[0].InstanceId)
	}
	tags = map[string]string{
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

	time.Sleep(2 * time.Minute)

	inst = &instOut.Instances[0]
	inst.PublicIpAddress = &publicIP
	return inst, nil
}

// PrepareBastionProxy will launch a bastion instance with squid proxy on the indicated zone and return the proxy url.
func (vpc *VPC) PrepareBastionProxy(zone string, keypairName string, privateKeyPath string) (proxyUrl string, err error) {
	encodeUserData := generateShellCommand()
	instance, err := vpc.LaunchBastion("", zone, encodeUserData, keypairName, privateKeyPath)
	if err != nil {
		log.LogError("Launch bastion failed")
		return "", err
	}

	privateKeyName := awsUtils.GetPrivateKeyName(privateKeyPath, keypairName)
	hostName := fmt.Sprintf("%s:%s", *instance.PublicIpAddress, CON.SSHPort)
	SSHExecuteCMDs, username, password, err := generateWriteSquidPasswordFileCommand()
	if err != nil {
		return "", err
	}
	for _, cmd := range SSHExecuteCMDs {
		_, err = Exec_CMD(CON.AWSInstanceUser, privateKeyName, hostName, cmd)
		if err != nil {
			log.LogError("SSH execute command failed")
			return "", err
		}
	}

	// construct proxy url
	proxy := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", *instance.PublicIpAddress, CON.SquidProxyPort),
		User:   url.UserPassword(username, password),
	}
	proxyUrl = proxy.String()
	return proxyUrl, nil
}

func (vpc *VPC) DestroyBastionProxy() error {
	filters := []map[string][]string{
		{
			"vpc-id": []string{
				vpc.VpcID,
			},
		},
	}
	filters = append(filters, map[string][]string{
		"tag:Name": {
			CON.BastionName,
		},
	})

	insts, err := vpc.AWSClient.ListInstances([]string{}, filters...)

	if err != nil {
		log.LogError("Error happened when list instances for vpc %s: %s", vpc.VpcID, err)
		return err
	}
	needTermination := []string{}
	keyPairNames := []string{}
	for _, inst := range insts {
		needTermination = append(needTermination, *inst.InstanceId)
		if inst.KeyName != nil {
			keyPairNames = append(keyPairNames, *inst.KeyName)
		}
	}
	err = vpc.AWSClient.TerminateInstances(needTermination, true, 20)
	if err != nil {
		log.LogError("Terminating instances %s meet error: %s", strings.Join(needTermination, ","), err)
	} else {
		log.LogInfo("Terminating instances %s successfully", strings.Join(needTermination, ","))
	}
	err = vpc.DeleteKeyPair(keyPairNames)
	if err != nil {
		log.LogError("Delete key pair %s meet error: %s", strings.Join(keyPairNames, ","), err)
	}
	needCleanGroups := []types.SecurityGroup{}
	securityGroups, err := vpc.AWSClient.ListSecurityGroups(vpc.VpcID)
	if err != nil {
		return err
	}
	for _, sg := range securityGroups {
		for _, tag := range sg.Tags {
			if *tag.Key == "Name" && *tag.Value == CON.AdditionalSecurityGroupName {
				needCleanGroups = append(needCleanGroups, sg)
			}
		}
	}
	for _, sg := range needCleanGroups {
		_, err = vpc.AWSClient.DeleteSecurityGroup(*sg.GroupId)
		if err != nil {
			return err
		}
	}
	return err
}

func generateBcryptPassword(plainPassword string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		log.LogError("Generate hashed password failed")
		return "", nil
	}
	log.LogInfo("Generate hashed password finished.")
	return string(hashedPassword), nil
}

func generateShellCommand() string {
	userData := fmt.Sprintf(`#!/bin/bash
		yum update -y
		sudo dnf install squid -y
		cd /etc/squid/
		sudo mv ./squid.conf ./squid.conf.bak
		sudo touch squid.conf
		echo http_port %s >> %s
		echo auth_param basic program /usr/lib64/squid/basic_ncsa_auth %s >> %s
		echo auth_param basic realm Squid Proxy Server >> %s
		echo acl authenticated proxy_auth REQUIRED >> %s
		echo http_access allow authenticated >> %s
		echo http_access deny all >> %s
		systemctl start squid
		systemctl enable squid`, CON.SquidProxyPort, CON.SquidConfigFilePath, CON.SquidPasswordFilePath,
		CON.SquidConfigFilePath, CON.SquidConfigFilePath, CON.SquidConfigFilePath, CON.SquidConfigFilePath,
		CON.SquidConfigFilePath)

	encodeUserData := base64.StdEncoding.EncodeToString([]byte(userData))
	log.LogInfo("Generate user data to creating squid proxy successfully.")

	return encodeUserData
}

func generateWriteSquidPasswordFileCommand() (SSHExecuteCMDs []string, username string,
	password string, err error) {
	username = utils.RandomLabel(5)
	password = utils.GeneratePassword(10)

	hashedPassword, err := generateBcryptPassword(password)
	if err != nil {
		log.LogError("Generate bcrypt password failed.")
		return []string{}, "", "", err
	}

	line := fmt.Sprintf("%s:%s\n", username, hashedPassword)
	remoteFilePath := CON.SquidPasswordFilePath

	createFileCMD := fmt.Sprintf("sudo touch %s", remoteFilePath)
	copyPasswordCMD := fmt.Sprintf("echo '%s' | sudo tee %s > /dev/null", line, remoteFilePath)
	SSHExecuteCMDs = []string{
		createFileCMD,
		copyPasswordCMD,
	}

	log.LogInfo("Generate write squid password file command finished.")
	return SSHExecuteCMDs, username, password, nil
}
