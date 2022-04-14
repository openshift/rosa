package ocm

import (
	"fmt"

	msv1 "github.com/openshift-online/ocm-sdk-go/servicemgmt/v1"
	"github.com/pkg/errors"
)

type CreateManagedServiceArgs struct {
	ServiceType string
	ClusterName string

	Parameters map[string]string
	Properties map[string]string

	AwsAccountID           string
	AwsRoleARN             string
	AwsSupportRoleARN      string
	AwsControlPlaneRoleARN string
	AwsWorkerRoleARN       string
	AwsRegion              string

	AwsOperatorIamRoleList []OperatorIAMRole

	MultiAZ           bool
	AvailabilityZones []string
	SubnetIDs         []string
}

func (c *Client) CreateManagedService(args CreateManagedServiceArgs) (*msv1.ManagedService, error) {

	operatorIamRoles := []*msv1.OperatorIAMRoleBuilder{}
	for _, operatorIAMRole := range args.AwsOperatorIamRoleList {
		operatorIamRoles = append(operatorIamRoles,
			msv1.NewOperatorIAMRole().
				Name(operatorIAMRole.Name).
				Namespace(operatorIAMRole.Namespace).
				RoleARN(operatorIAMRole.RoleARN))
	}

	parameters := []*msv1.ServiceParameterBuilder{}
	for id, val := range args.Parameters {
		parameters = append(parameters,
			msv1.NewServiceParameter().ID(id).Value(val))
	}

	service, err := msv1.NewManagedService().
		Service(args.ServiceType).
		Parameters(parameters...).
		Cluster(
			msv1.NewCluster().
				Name(args.ClusterName).
				Properties(args.Properties).
				Region(
					msv1.NewCloudRegion().
						ID(args.AwsRegion)).
				MultiAZ(args.MultiAZ).
				AWS(
					msv1.NewAWS().
						STS(msv1.NewSTS().
							RoleARN(args.AwsRoleARN).
							SupportRoleARN(args.AwsSupportRoleARN).
							InstanceIAMRoles(msv1.NewInstanceIAMRoles().
								MasterRoleARN(args.AwsControlPlaneRoleARN).
								WorkerRoleARN(args.AwsWorkerRoleARN)).
							OperatorIAMRoles(operatorIamRoles...)).
						AccountID(args.AwsAccountID).
						SubnetIDs(args.SubnetIDs...)).
				Nodes(msv1.NewClusterNodes().
					AvailabilityZones(args.AvailabilityZones...))).
		Build()

	if err != nil {
		return nil, errors.Wrap(err, "Failed to create Managed Service call")
	}

	serviceCall, err := c.ocm.ServiceMgmt().V1().Services().
		Add().
		Body(service).
		Send()
	if err != nil {
		return nil, handleErr(serviceCall.Error(), err)
	}

	return serviceCall.Body(), nil
}

func (c *Client) ListManagedServices(count int) (*msv1.ManagedServiceList, error) {
	if count < 0 {
		err := errors.Errorf("Invalid services count")
		return nil, err
	}

	response, err := c.ocm.ServiceMgmt().V1().Services().List().Send()
	if err != nil {
		fmt.Printf("%s", err)
		err := errors.Errorf("Cannot retrieve services list")
		return nil, err
	}
	return response.Items(), nil

}

type DescribeManagedServiceArgs struct {
	ID string
}

func (c *Client) GetManagedService(args DescribeManagedServiceArgs) (*msv1.ManagedService, error) {
	response, err := c.ocm.ServiceMgmt().V1().Services().Service(args.ID).Get().Send()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get managed service with id %s:", args.ID)
	}
	return response.Body(), nil
}

type DeleteManagedServiceArgs struct {
	ID string
}

func (c *Client) DeleteManagedService(args DeleteManagedServiceArgs) (*msv1.ManagedServiceDeleteResponse, error) {
	deleteResponse, err := c.ocm.ServiceMgmt().V1().Services().
		Service(args.ID).
		Delete().
		Send()
	if err != nil {
		return nil, handleErr(deleteResponse.Error(), err)
	}

	return deleteResponse, nil
}
