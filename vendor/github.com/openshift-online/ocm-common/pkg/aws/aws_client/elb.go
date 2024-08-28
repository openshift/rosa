package aws_client

import (
	"context"

	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"

	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (client *AWSClient) DescribeLoadBalancers(vpcID string) ([]elbtypes.LoadBalancer, error) {

	listenedELB := []elbtypes.LoadBalancer{}
	input := &elb.DescribeLoadBalancersInput{}
	resp, err := client.ElbClient.DescribeLoadBalancers(context.TODO(), input)
	if err != nil {
		return nil, err
	}
	for _, lb := range resp.LoadBalancers {
		if *lb.VpcId == vpcID {
			log.LogInfo("Got load balancer %s", *lb.LoadBalancerName)
			listenedELB = append(listenedELB, lb)
		}
	}

	return listenedELB, err
}

func (client *AWSClient) DeleteELB(ELB elbtypes.LoadBalancer) error {
	log.LogInfo("Going to delete ELB %s", *ELB.LoadBalancerName)

	deleteELBInput := &elb.DeleteLoadBalancerInput{
		// LoadBalancerArn: ELB.LoadBalancerArn,
		LoadBalancerArn: ELB.LoadBalancerArn,
	}
	_, err := client.ElbClient.DeleteLoadBalancer(context.TODO(), deleteELBInput)
	return err
}
