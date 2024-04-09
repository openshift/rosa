package securitygroups

import (
	"fmt"
	"os"
	"strconv"

	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	. "github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	additionalComputeSecurityGroupIdsFlag      = "additional-compute-security-group-ids"
	additionalInfraSecurityGroupIdsFlag        = "additional-infra-security-group-ids"
	additionalControlPlaneSecurityGroupIdsFlag = "additional-control-plane-security-group-ids"
	securityGroupIdsFlag                       = "additional-security-group-ids"

	ComputeKind      = "Compute"
	InfraKind        = "Infra"
	ControlPlaneKind = "Control Plane"
	MachinePoolKind  = "Machine Pool"
)

var (
	SgKindFlagMap = map[string]string{
		ComputeKind:      additionalComputeSecurityGroupIdsFlag,
		InfraKind:        additionalInfraSecurityGroupIdsFlag,
		ControlPlaneKind: additionalControlPlaneSecurityGroupIdsFlag,
		MachinePoolKind:  securityGroupIdsFlag,
	}
	ComputeSecurityGroupFlag      = SgKindFlagMap[ComputeKind]
	InfraSecurityGroupFlag        = SgKindFlagMap[InfraKind]
	ControlPlaneSecurityGroupFlag = SgKindFlagMap[ControlPlaneKind]
	MachinePoolSecurityGroupFlag  = SgKindFlagMap[MachinePoolKind]
)

func GetSecurityGroupIds(r *rosa.Runtime, cmd *cobra.Command,
	targetVpcId string, kind string, id string) []string {
	possibleSgs, err := r.AWSClient.GetSecurityGroupIds(targetVpcId)
	if err != nil {
		r.Reporter.Errorf("There was a problem retrieving security groups for VPC '%s': %v", targetVpcId, err)
		os.Exit(1)
	}
	securityGroupIds := []string{}
	if len(possibleSgs) > 0 {
		options := []string{}
		for _, sg := range possibleSgs {
			if isValidSecurityGroup(sg, id) {
				options = append(options, aws.SetSecurityGroupOption(sg))
			}
		}
		// No available security groups.
		if len(options) == 0 {
			return securityGroupIds
		}

		securityGroupIds, err = GetMultipleOptions(Input{
			Question: fmt.Sprintf("Additional '%s' Security Group IDs", kind),
			Help:     cmd.Flags().Lookup(SgKindFlagMap[kind]).Usage,
			Required: false,
			Options:  options,
		})
		if err != nil {
			r.Reporter.Errorf("Expected valid Security Group IDs: %s", err)
			os.Exit(1)
		}
		for i, sg := range securityGroupIds {
			securityGroupIds[i] = aws.ParseOption(sg)
		}
	}
	return securityGroupIds
}

func isValidSecurityGroup(sg types.SecurityGroup, id string) bool {
	if aws.Ec2ResourceHasTag(sg.Tags, tags.RedHatManaged, strconv.FormatBool(true)) {
		return false
	}
	if awsSdk.ToString(sg.GroupName) == "default" {
		return false
	}
	if id != "" {
		if aws.Ec2ResourceHasTag(sg.Tags, fmt.Sprintf("kubernetes.io/cluster/%s", id), "owned") {
			return false
		}
	}

	return true
}
