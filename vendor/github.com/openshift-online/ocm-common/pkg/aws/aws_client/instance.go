package aws_client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"

	"github.com/openshift-online/ocm-common/pkg/log"
)

func (client *AWSClient) LaunchInstance(subnetID string, imageID string, count int, instanceType string, keyName string, securityGroupIds []string, wait bool) (*ec2.RunInstancesOutput, error) {
	input := &ec2.RunInstancesInput{
		ImageId:          aws.String(imageID),
		MinCount:         aws.Int32(int32(count)),
		MaxCount:         aws.Int32(int32(count)),
		InstanceType:     types.InstanceType(instanceType),
		KeyName:          aws.String(keyName),
		SecurityGroupIds: securityGroupIds,
		SubnetId:         &subnetID,
	}
	output, err := client.Ec2Client.RunInstances(context.TODO(), input)
	if wait && err == nil {
		instanceIDs := []string{}
		for _, instance := range output.Instances {
			instanceIDs = append(instanceIDs, *instance.InstanceId)
		}
		log.LogInfo("Waiting for below instances ready: %s", strings.Join(instanceIDs, "，"))
		_, err = client.WaitForInstancesRunning(instanceIDs, 10)
		if err != nil {
			log.LogError("Error happened for instance running: %s", err)
		} else {
			log.LogInfo("All instances running")
		}
	}
	return output, err
}

// ListInstance pass parameter like
// map[string][]string{"vpc-id":[]string{"<id>" }}, map[string][]string{"tag:Name":[]string{"<value>" }}
// instanceIDs can be empty. And if you would like to get more info from the instances like security groups, it should be set
func (client *AWSClient) ListInstances(instanceIDs []string, filters ...map[string][]string) ([]types.Instance, error) {
	FilterInput := []types.Filter{}
	for _, filter := range filters {
		for k, v := range filter {
			awsFilter := types.Filter{
				Name:   &k,
				Values: v,
			}
			FilterInput = append(FilterInput, awsFilter)
		}
	}
	getInstanceInput := &ec2.DescribeInstancesInput{
		Filters: FilterInput,
	}
	if len(instanceIDs) != 0 {
		getInstanceInput.InstanceIds = instanceIDs
	}
	resp, err := client.EC2().DescribeInstances(context.TODO(), getInstanceInput)
	if err != nil {
		log.LogError("List instances failed with filters %v: %s", filters, err)
	}
	var instances []types.Instance
	for _, reserv := range resp.Reservations {
		instances = append(instances, reserv.Instances...)
	}
	return instances, err
}

func (client *AWSClient) WaitForInstanceReady(instanceID string, timeout time.Duration) error {
	instanceIDs := []string{
		instanceID,
	}
	log.LogInfo("Waiting for below instances ready: %s ", strings.Join(instanceIDs, "|"))
	_, err := client.WaitForInstancesRunning(instanceIDs, 10)
	return err
}

func (client *AWSClient) CheckInstanceState(instanceIDs ...string) (*ec2.DescribeInstanceStatusOutput, error) {
	log.LogInfo("Check instances status of %s", strings.Join(instanceIDs, ","))
	includeAll := true
	input := &ec2.DescribeInstanceStatusInput{
		InstanceIds:         instanceIDs,
		IncludeAllInstances: &includeAll,
	}
	output, err := client.Ec2Client.DescribeInstanceStatus(context.TODO(), input)
	return output, err
}

// timeout indicates the minutes
func (client *AWSClient) WaitForInstancesRunning(instanceIDs []string, timeout time.Duration) (allRunning bool, err error) {
	startTime := time.Now()

	for time.Now().Before(startTime.Add(timeout * time.Minute)) {
		allRunning = true
		output, err := client.CheckInstanceState(instanceIDs...)
		if err != nil {
			log.LogError("Error happened when describe instant status: %s", strings.Join(instanceIDs, ","))
			return false, err
		}
		if len(output.InstanceStatuses) == 0 {
			log.LogWarning("Instance status description for %s is 0", strings.Join(instanceIDs, ","))
		}
		for _, ins := range output.InstanceStatuses {
			log.LogInfo("Instance ID %s is in status of %s", *ins.InstanceId, ins.InstanceStatus.Status)
			log.LogInfo("Instance ID %s is in state of %s", *ins.InstanceId, ins.InstanceState.Name)
			if ins.InstanceState.Name != types.InstanceStateNameRunning && ins.InstanceStatus.Status != types.SummaryStatusOk {
				allRunning = false
			}

		}
		if allRunning {
			return true, nil
		}
		time.Sleep(time.Minute)
	}
	err = fmt.Errorf("timeout for waiting instances running")
	return
}
func (client *AWSClient) WaitForInstancesTerminated(instanceIDs []string, timeout time.Duration) (allTerminated bool, err error) {
	startTime := time.Now()
	for time.Now().Before(startTime.Add(timeout * time.Minute)) {
		allTerminated = true
		output, err := client.CheckInstanceState(instanceIDs...)
		if err != nil {
			log.LogError("Error happened when describe instant status: %s", strings.Join(instanceIDs, ","))
			return false, err
		}
		if len(output.InstanceStatuses) == 0 {
			log.LogWarning("Instance status description for %s is 0", strings.Join(instanceIDs, ","))
		}
		for _, ins := range output.InstanceStatuses {
			log.LogInfo("Instance ID %s is in status of %s", *ins.InstanceId, ins.InstanceStatus.Status)
			log.LogInfo("Instance ID %s is in state of %s", *ins.InstanceId, ins.InstanceState.Name)
			if ins.InstanceState.Name != types.InstanceStateNameTerminated {
				allTerminated = false
			}

		}
		if allTerminated {
			return true, nil
		}
		time.Sleep(time.Minute)
	}
	err = fmt.Errorf("timeout for waiting instances terminated")
	return

}

// Search instance types for specified region/availability zones
func (client *AWSClient) ListAvaliableInstanceTypesForRegion(region string, availabilityZones ...string) ([]string, error) {
	var params *ec2.DescribeInstanceTypeOfferingsInput
	if len(availabilityZones) > 0 {
		params = &ec2.DescribeInstanceTypeOfferingsInput{
			Filters:      []types.Filter{{Name: aws.String("location"), Values: availabilityZones}},
			LocationType: types.LocationTypeAvailabilityZone,
		}
	} else {
		params = &ec2.DescribeInstanceTypeOfferingsInput{
			Filters: []types.Filter{{Name: aws.String("location"), Values: []string{region}}},
		}
	}
	var instanceTypes []types.InstanceTypeOffering
	paginator := ec2.NewDescribeInstanceTypeOfferingsPaginator(client.Ec2Client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		instanceTypes = append(instanceTypes, page.InstanceTypeOfferings...)
	}
	machineTypeList := make([]string, len(instanceTypes))
	for i, v := range instanceTypes {
		machineTypeList[i] = string(v.InstanceType)
	}
	return machineTypeList, nil
}

// List avaliablezone for specific region
// zone type are: local-zone/availability-zone/wavelength-zone
func (client *AWSClient) ListAvaliableZonesForRegion(region string, zoneType string) ([]string, error) {
	var zones []string
	availabilityZones, err := client.Ec2Client.DescribeAvailabilityZones(context.TODO(), &ec2.DescribeAvailabilityZonesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("region-name"),
				Values: []string{region},
			},
			{
				Name:   aws.String("zone-type"),
				Values: []string{zoneType},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(availabilityZones.AvailabilityZones) < 1 {
		return zones, nil
	}

	for _, v := range availabilityZones.AvailabilityZones {
		zones = append(zones, *v.ZoneName)
	}
	return zones, nil
}
func (client *AWSClient) TerminateInstances(instanceIDs []string, wait bool, timeout time.Duration) error {
	if len(instanceIDs) == 0 {
		log.LogInfo("Got no instances to terminate.")
		return nil
	}
	terminateInput := &ec2.TerminateInstancesInput{
		InstanceIds: instanceIDs,
	}
	_, err := client.EC2().TerminateInstances(context.TODO(), terminateInput)
	if err != nil {
		log.LogError("Error happens when terminate instances %s : %s", strings.Join(instanceIDs, ","), err)
		return err
	} else {
		log.LogInfo("Terminate instances %s successfully", strings.Join(instanceIDs, ","))
	}
	if wait {
		err = client.WaitForInstanceTerminated(instanceIDs, timeout)
		if err != nil {
			log.LogError("Waiting for  instances %s termination timeout %s ", strings.Join(instanceIDs, ","), err)
			return err
		}

	}
	return nil
}

func (client *AWSClient) WaitForInstanceTerminated(instanceIDs []string, timeout time.Duration) error {
	log.LogInfo("Waiting for below instances terminated: %s ", strings.Join(instanceIDs, ","))
	_, err := client.WaitForInstancesTerminated(instanceIDs, timeout)
	return err
}

func (client *AWSClient) GetTagsOfInstanceProfile(instanceProfileName string) ([]iamtypes.Tag, error) {
	input := &iam.ListInstanceProfileTagsInput{
		InstanceProfileName: &instanceProfileName,
	}
	resp, err := client.IamClient.ListInstanceProfileTags(context.TODO(), input)
	if err != nil {
		return nil, err
	}
	tags := resp.Tags
	return tags, err
}

func GetInstanceName(instance *types.Instance) string {
	tags := instance.Tags
	for _, tag := range tags {
		if *tag.Key == "Name" {
			return *tag.Value
		}
	}
	return ""
}

// GetInstancesByInfraID will return the instances with tag tag:kubernetes.io/cluster/<infraID>
func (client *AWSClient) GetInstancesByInfraID(infraID string) ([]types.Instance, error) {
	filter := types.Filter{
		Name: aws.String("tag:kubernetes.io/cluster/" + infraID),
		Values: []string{
			"owned",
		},
	}
	output, err := client.Ec2Client.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			filter,
		},
		MaxResults: aws.Int32(100),
	})
	if err != nil {
		return nil, err
	}
	var instances []types.Instance
	for _, reservation := range output.Reservations {
		instances = append(instances, reservation.Instances...)
	}
	return instances, err
}

// GetInstancesByNodePoolID will return the instances with tag tag:api.openshift.com/nodepool-ocm:<nodepool_id>
func (client *AWSClient) GetInstancesByNodePoolID(nodePoolID string, clusterID string) ([]types.Instance, error) {
	filter1 := types.Filter{
		Name: aws.String("tag:api.openshift.com/nodepool-ocm"),
		Values: []string{
			nodePoolID,
		},
	}
	filter2 := types.Filter{
		Name: aws.String("tag:api.openshift.com/id"),
		Values: []string{
			clusterID,
		},
	}
	output, err := client.Ec2Client.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			filter1,
			filter2,
		},
		MaxResults: aws.Int32(100),
	})
	if err != nil {
		return nil, err
	}
	var instances []types.Instance
	for _, reservation := range output.Reservations {
		instances = append(instances, reservation.Instances...)
	}
	return instances, err
}

func (client *AWSClient) ListAvaliableRegionsFromAWS() ([]types.Region, error) {
	optInStatus := "opt-in-status"
	optInNotRequired := "opt-in-not-required"
	optIn := "opted-in"
	filter := types.Filter{Name: &optInStatus, Values: []string{optInNotRequired, optIn}}

	output, err := client.Ec2Client.DescribeRegions(context.TODO(), &ec2.DescribeRegionsInput{
		Filters: []types.Filter{
			filter,
		},
	})

	return output.Regions, err
}
