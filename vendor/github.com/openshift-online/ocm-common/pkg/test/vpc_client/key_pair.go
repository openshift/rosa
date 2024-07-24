package vpc_client

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (vpc *VPC) CreateKeyPair(keyName string) (*ec2.CreateKeyPairOutput, error) {
	output, err := vpc.AWSClient.CreateKeyPair(keyName)
	if err != nil {
		log.LogError("Create key pair meets error %s", err.Error())
		return nil, err
	}
	log.LogInfo("Create key pair %v successfully\n", *output.KeyPairId)

	return output, nil
}

func (vpc *VPC) DeleteKeyPair(keyNames []string) error {
	for _, key := range keyNames {
		_, err := vpc.AWSClient.DeleteKeyPair(key)
		if err != nil {
			log.LogError("Delete key pair meets error %s", err.Error())
			return err
		}
	}
	log.LogInfo("Delete key pair successfully")
	return nil
}
