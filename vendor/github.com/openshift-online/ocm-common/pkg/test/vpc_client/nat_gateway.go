package vpc_client

import (
	"sync"

	"github.com/openshift-online/ocm-common/pkg/log"
)

func (vpc *VPC) DeleteVPCNatGateways(vpcID string) error {

	var delERR error
	natGateways, err := vpc.AWSClient.ListNatGateWays(vpcID)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	allocationId := ""
	for _, ngw := range natGateways {
		if len(ngw.NatGatewayAddresses) > 0 {
			for _, add := range ngw.NatGatewayAddresses {
				allocationId = *add.AllocationId
			}
		}
		log.LogInfo("Deleting nat gateway %s", *ngw.NatGatewayId)
		wg.Add(1)
		go func(gateWayID string) {
			defer wg.Done()
			_, err = vpc.AWSClient.DeleteNatGateway(gateWayID, 180)
			if err != nil {
				delERR = err
			}
		}(*ngw.NatGatewayId)
	}
	wg.Wait()
	if delERR != nil {
		return delERR
	}
	if allocationId != "" {
		err = vpc.AWSClient.ReleaseAddressWithAllocationID(allocationId)
	}
	return err
}
