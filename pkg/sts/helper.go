package sts

import (
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/aws"
)

func GetNewOperatorAdded(cluster *cmv1.Cluster) map[string]aws.Operator {
	newCredRequest := make(map[string]aws.Operator)
	for credRequest,operator := range aws.CredentialRequests{
		exists:=false
		for _, role := range cluster.AWS().STS().OperatorIAMRoles() {
			if role.Namespace() == operator.Namespace && role.Name() == operator.Name {
				exists = true
			}
		}
		if !exists{
			newCredRequest[credRequest]= operator
		}
	}
	return newCredRequest
}

func IsNewOperatorAdded(version string) bool {
	for _, operator := range aws.CredentialRequests {
		if operator.Version == version {
			return true
		}
	}
	return false
}
