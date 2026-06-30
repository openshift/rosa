package vpc_client

import (
	"errors"
	"fmt"

	"github.com/openshift-online/ocm-common/pkg/log"
)

func (vpc *VPC) DeleteVPCNetworkInterfaces() error {
	networkInterfaces, err := vpc.AWSClient.DescribeNetWorkInterface(vpc.VpcID)
	if err != nil {
		return err
	}

	var errs []error
	for _, networkInterface := range networkInterfaces {
		if networkInterface.Attachment != nil && networkInterface.Attachment.AttachmentId != nil {
			log.LogInfo("Detaching network interface %s", *networkInterface.NetworkInterfaceId)
			if err := vpc.AWSClient.DetachNetworkInterface(*networkInterface.Attachment.AttachmentId, true); err != nil {
				log.LogError("Detach network interface %s failed: %s", *networkInterface.NetworkInterfaceId, err)
				errs = append(errs, fmt.Errorf("detach ENI %s: %w", *networkInterface.NetworkInterfaceId, err))
				continue
			}
		}
		if err := vpc.AWSClient.DeleteNetworkInterface(networkInterface); err != nil {
			errs = append(errs, fmt.Errorf("delete ENI %s: %w", *networkInterface.NetworkInterfaceId, err))
		}
	}

	return errors.Join(errs...)
}
