package handler

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"

	ClusterConfigure "github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/constants"
	"github.com/openshift/rosa/tests/utils/helper"
	"github.com/openshift/rosa/tests/utils/log"
)

const rosaAutoConfirmFlag = "-y" // skip interactive rosa prompts in FVT

func (ch *clusterHandler) generateRegionalPlatformCreateFlags(clusterName string) ([]string, error) {
	flags := []string{rosaAutoConfirmFlag}

	if v, ok := resolvePlatformAPIVersion(ch.profile.Version); ok {
		flags = append(flags, "--version", v)
		ch.clusterConfig.Version = &ClusterConfigure.Version{RawID: v, VersionRequirement: v}
	}

	if r := ch.profile.Region; r != "" {
		flags = append(flags, "--region", r)
		ch.clusterConfig.Region = r
	}

	prefix := helper.TrimNameByLength(clusterName, constants.MaxRolePrefixLength)
	flags = append(flags, "--operator-roles-prefix", prefix)
	ch.clusterConfig.Sts = true
	ch.clusterConfig.Aws = &ClusterConfigure.AWS{Sts: ClusterConfigure.Sts{OperatorRolesPrefix: prefix}}

	if ch.profile.ClusterConfig.BYOVPC {
		subnetIDs, err := prepareBYOVPCSubnets(ch.resourcesHandler, clusterName, ch.profile, ch.clusterConfig)
		if err != nil {
			return flags, err
		}
		flags = append(flags, "--subnet-ids", subnetIDs)
		if waitErr := waitForSubnetsVisibleFromAccount(
			ch.resourcesHandler, helper.RemoveFromStringSlice(strings.Split(subnetIDs, ","), ""),
		); waitErr != nil {
			return flags, waitErr
		}
		if err := preparePlatformAPIPreCreateInfra(ch.resourcesHandler, clusterName); err != nil {
			return flags, err
		}
	}

	pc := ch.profile.ClusterConfig
	if pc.HCP {
		flags = append(flags, "--hosted-cp")
	}
	if pc.MultiAZ {
		flags = append(flags, "--multi-az")
		ch.clusterConfig.MultiAZ = pc.MultiAZ
	}
	ch.clusterConfig.Nodes = &ClusterConfigure.Nodes{}
	if t := pc.InstanceType; t != "" {
		flags = append(flags, "--compute-machine-type", t)
		ch.clusterConfig.Nodes.ComputeInstanceType = t
	}
	return flags, nil
}

func resolvePlatformAPIVersion(profileVersion string) (string, bool) {
	if v := os.Getenv("HYPERFLEET_VERSION"); v != "" {
		return v, true
	}

	if profileVersion != "" && constants.VersionRawPattern.MatchString(profileVersion) {
		return profileVersion, true
	}

	if profileVersion != "" {
		log.Logger.Infof("Skipping OCM version lookup for %q; Platform API resolves default (set HYPERFLEET_VERSION to pin)", profileVersion)
	}
	return "", false
}

func prepareBYOVPCSubnets(rh *resourcesHandler, name string, profile *Profile, cfg *ClusterConfigure.ClusterConfig) (string, error) {
	cidr := constants.DefaultVPCCIDRValue
	if profile.ClusterConfig.NetworkingSet {
		cidr = cfg.Networking.MachineCIDR
	}

	if _, err := rh.PrepareVPC(helper.TrimNameByLength(name, 20), cidr, false, profile.ClusterConfig.SharedVPC); err != nil {
		return "", err
	}

	zones := helper.RemoveFromStringSlice(strings.Split(profile.ClusterConfig.Zones, ","), "")
	subnets, err := rh.PrepareSubnets(zones, profile.ClusterConfig.MultiAZ)
	if err != nil {
		return "", err
	}

	cfg.Subnets = &ClusterConfigure.Subnets{
		PrivateSubnetIds: strings.Join(subnets["private"], ","),
		PublicSubnetIds:  strings.Join(subnets["public"], ","),
	}

	return strings.Join(append(subnets["private"], subnets["public"]...), ","), nil
}

func preparePlatformAPIPreCreateInfra(rh *resourcesHandler, clusterName string) error {
	if rh.vpc == nil {
		return fmt.Errorf("VPC required before Platform API pre-create infrastructure")
	}
	if _, err := rh.preparePlatformAPIHostedZone(clusterName+".hypershift.local", rh.vpc.VpcID); err != nil {
		return err
	}
	return rh.preparePlatformAPIWorkerSecurityGroup(clusterName)
}

func (ch *clusterHandler) waitForRegionalPlatformClusterReady(timeoutMin int) error {
	clusterKey := ch.clusterDetail.ClusterName
	if clusterKey == "" {
		clusterKey = ch.profile.ClusterConfig.Name
	}
	if clusterKey == "" {
		return errors.New("no cluster name defined to wait for")
	}

	defer func() {
		log.Logger.Info("Going to record the necessary information")
		ch.saveToFile()
	}()

	clusterService := ch.rosaClient.Cluster
	endTime := time.Now().Add(time.Duration(timeoutMin) * time.Minute)
	for time.Now().Before(endTime) {
		description, err := clusterService.DescribeClusterAndReflect(clusterKey)
		if err != nil {
			return err
		}
		ch.clusterDetail.APIURL = description.APIURL
		ch.clusterDetail.ConsoleURL = description.ConsoleURL
		ch.clusterDetail.InfraID = description.InfraID

		phase := strings.TrimSpace(description.State)
		switch phase {
		case string(v1alpha1.ClusterPhaseReady):
			log.Logger.Infof("Cluster %s is ready now.", clusterKey)
			return nil
		case string(v1alpha1.ClusterPhaseDeleting):
			return fmt.Errorf("cluster %s is %s now. Cannot wait for it ready", clusterKey, phase)
		case string(v1alpha1.ClusterPhaseWaitingForPlacement), string(v1alpha1.ClusterPhaseProvisioning), "":
			log.Logger.Infof("Cluster %s phase is %q, waiting for Ready", clusterKey, phase)
			time.Sleep(2 * time.Minute)
		default:
			return fmt.Errorf("unknown cluster phase %q", phase)
		}
	}

	return fmt.Errorf("timeout for cluster ready waiting after %d mins", timeoutMin)
}
