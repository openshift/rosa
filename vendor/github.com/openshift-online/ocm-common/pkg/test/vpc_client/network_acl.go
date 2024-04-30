package vpc_client

import "github.com/openshift-online/ocm-common/pkg/log"

func (vpc *VPC) AddSimplyDenyRuleToNetworkACL(port int32, ruleNumber int32) error {
	err := vpc.AddNetworkACLRules(true, "6", "deny", ruleNumber, port, port, "0.0.0.0/0")
	return err
}

func (vpc *VPC) AddNetworkACLRules(egress bool, protocol string, ruleAction string, ruleNumber int32, fromPort int32, toPort int32, cidrBlock string) error {
	acls, err := vpc.AWSClient.ListNetWorkAcls(vpc.VpcID)
	if err != nil {
		return err
	}
	networkAclId := *acls[0].NetworkAclId
	log.LogInfo("Find Network ACL" + networkAclId)
	_, err = vpc.AWSClient.AddNetworkAclEntry(networkAclId, egress, protocol, ruleAction, ruleNumber, fromPort, toPort, cidrBlock)
	return err
}

func (vpc *VPC) DeleteNetworkACLRules(egress bool, ruleNumber int32) error {
	acls, err := vpc.AWSClient.ListNetWorkAcls(vpc.VpcID)
	if err != nil {
		return err
	}
	networkAclId := *acls[0].NetworkAclId
	log.LogInfo("Find Network ACL" + networkAclId)

	_, err = vpc.AWSClient.DeleteNetworkAclEntry(networkAclId, egress, ruleNumber)
	return err
}
