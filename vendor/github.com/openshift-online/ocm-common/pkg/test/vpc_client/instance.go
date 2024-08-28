package vpc_client

import (
	"strings"

	CON "github.com/openshift-online/ocm-common/pkg/aws/consts"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (vpc *VPC) TerminateVPCInstances(nonClusterOnly bool) error {
	filters := []map[string][]string{
		{
			"vpc-id": []string{
				vpc.VpcID,
			},
		},
	}
	if nonClusterOnly {
		filters = append(filters, map[string][]string{
			"tag:Name": {
				CON.ProxyName,
				CON.BastionName,
			},
		})
	}
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
	return err

}
