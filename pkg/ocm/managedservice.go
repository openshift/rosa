package ocm

import (
	msv1 "github.com/openshift-online/ocm-sdk-go/servicemgmt/v1"
	"github.com/pkg/errors"
)

func (c *Client) CreateService() (*msv1.ManagedService, error) {

	service, err := msv1.NewManagedService().
		Service("badabing!").
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
