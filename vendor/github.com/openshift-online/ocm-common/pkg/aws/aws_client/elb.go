package aws_client

import (
	"context"

	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"

	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing/types"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (client *AWSClient) DescribeLoadBalancers(vpcID string) ([]elbtypes.LoadBalancerDescription, error) {

	listenedELB := []elbtypes.LoadBalancerDescription{}
	input := &elb.DescribeLoadBalancersInput{}
	resp, err := client.ElbClient.DescribeLoadBalancers(context.TODO(), input)
	if err != nil {
		return nil, err
	}
	// for _, lb := range resp.LoadBalancers {
	for _, lb := range resp.LoadBalancerDescriptions {

		// if *lb.VpcId == vpcID {
		if *lb.VPCId == vpcID {
			log.LogInfo("Got load balancer %s", *lb.LoadBalancerName)
			listenedELB = append(listenedELB, lb)
		}
	}

	return listenedELB, err
}

func (client *AWSClient) DeleteELB(ELB elbtypes.LoadBalancerDescription) error {
	log.LogInfo("Goint to delete ELB %s", *ELB.LoadBalancerName)

	deleteELBInput := &elb.DeleteLoadBalancerInput{
		// LoadBalancerArn: ELB.LoadBalancerArn,
		LoadBalancerName: ELB.LoadBalancerName,
	}
	_, err := client.ElbClient.DeleteLoadBalancer(context.TODO(), deleteELBInput)
	return err
}
