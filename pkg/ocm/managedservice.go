package ocm

import (
	msv1 "github.com/openshift-online/ocm-sdk-go/servicemgmt/v1"
	"github.com/pkg/errors"
)

type CreateManagedServiceArgs struct {
	ServiceName string
	ClusterName string

	AwsAccountID       string
	AwsAccessKeyID     string
	AwsSecretAccessKey string
	AwsRegion          string
}

func (c *Client) CreateManagedService(args CreateManagedServiceArgs) (*msv1.ManagedService, error) {

	service, err := msv1.NewManagedService().
		Service(args.ServiceName).
		Cluster(
			msv1.NewCluster().
				Name(args.ClusterName).
				Region(
					msv1.NewCloudRegion().
						ID(args.AwsRegion)).
				AWS(
					msv1.NewAWS().
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
