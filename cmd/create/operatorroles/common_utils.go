package operatorroles

import (
	"fmt"

	"github.com/openshift/rosa/pkg/aws"
)

func computePolicyARN(accountID string, prefix string, namespace string, name string, path string) string {
	if prefix == "" {
		prefix = aws.DefaultPrefix
	}
	policy := fmt.Sprintf("%s-%s-%s", prefix, namespace, name)
	if len(policy) > 64 {
		policy = policy[0:64]
	}
	if path != "" {
		return fmt.Sprintf("arn:%s:iam::%s:policy%s%s", aws.GetPartition(), accountID, path, policy)
	}
	return fmt.Sprintf("arn:%s:iam::%s:policy/%s", aws.GetPartition(), accountID, policy)
}
