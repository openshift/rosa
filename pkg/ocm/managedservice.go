package ocm

import (
	msv1 "github.com/openshift-online/ocm-sdk-go/servicemgmt/v1"
	"github.com/pkg/errors"
)

type CreateManagedServiceArgs struct {
	ServiceName string
	ClusterName string

	AwsAccountID           string
	AwsAccessKeyID         string
	AwsSecretAccessKey     string
	AwsRoleARN             string
	AwsSupportRoleARN      string
	AwsControlPlaneRoleARN string
	AwsWorkerRoleARN       string
	AwsRegion              string

	AwsOperatorIamRoleList []OperatorIAMRole
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

	service, err := msv1.NewManagedService().
		Service(args.ServiceName).
		Parameters(
			msv1.NewServiceParameter().
				ID("has-external-resources").
				Value("true"),
			msv1.NewServiceParameter().
				ID("parameter-with-requirements").
				Value("1"),
		).
		Cluster(
			msv1.NewCluster().
				Name(args.ClusterName).
				Region(
					msv1.NewCloudRegion().
						ID(args.AwsRegion)).
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
						AccessKeyID(args.AwsAccessKeyID).
						SecretAccessKey(args.AwsSecretAccessKey))).
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
