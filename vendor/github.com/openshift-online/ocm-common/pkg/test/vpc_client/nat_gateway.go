package vpc_client

import (
	"errors"
	"sync"

	awserrors "github.com/openshift-online/ocm-common/pkg/aws/errors"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (vpc *VPC) DeleteVPCNatGateways(vpcID string) error {
	natGateways, err := vpc.AWSClient.ListNatGateways(vpcID)
	if err != nil {
		return err
	}

	var allocationIDs []string
	for _, ngw := range natGateways {
		for _, addr := range ngw.NatGatewayAddresses {
			if addr.AllocationId != nil {
				allocationIDs = append(allocationIDs, *addr.AllocationId)
			}
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var delErrs []error
	for _, ngw := range natGateways {
		log.LogInfo("Deleting nat gateway %s", *ngw.NatGatewayId)
		wg.Add(1)
		go func(gateWayID string) {
			defer wg.Done()
			_, err := vpc.AWSClient.DeleteNatGateway(gateWayID, 180)
			if err != nil {
				mu.Lock()
				delErrs = append(delErrs, err)
				mu.Unlock()
			}
		}(*ngw.NatGatewayId)
	}
	wg.Wait()

	var releaseErrs []error
	for _, allocID := range allocationIDs {
		log.LogInfo("Releasing EIP allocation %s", allocID)
		err := vpc.AWSClient.ReleaseAddressWithAllocationID(allocID)
		if err != nil && !awserrors.IsErrorCode(err, awserrors.InvalidAllocationID) {
			releaseErrs = append(releaseErrs, err)
		}
	}

	allErrs := append(delErrs, releaseErrs...)
	return errors.Join(allErrs...)
}
