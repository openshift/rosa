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

func (ch *clusterHandler) generateHyperfleetCreateFlags() ([]string, error) {
	clusterName := strings.TrimSpace(os.Getenv("CLUSTER_NAME"))
	if clusterName == "" || len(clusterName) > 18 {
		return nil, fmt.Errorf("CLUSTER_NAME is required and must be ≤18 chars")
	}
	ch.profile.ClusterConfig.Name = clusterName
	ch.clusterConfig.Name = clusterName

	flags := []string{rosaAutoConfirmFlag}

	if v, ok := resolvePlatformAPIVersion(ch.profile.Version); ok {
		flags = append(flags, "--version", v)
		ch.clusterConfig.Version = &ClusterConfigure.Version{RawID: v, VersionRequirement: v}
	}
	if ch.profile.ChannelGroup != "" {
		if ch.clusterConfig.Version == nil {
			ch.clusterConfig.Version = &ClusterConfigure.Version{}
		}
		ch.clusterConfig.Version.ChannelGroup = ch.profile.ChannelGroup
	}
	if ch.profile.Version != "" && ch.clusterConfig.Version != nil {
		ch.clusterConfig.Version.VersionRequirement = ch.profile.Version
	}

	if r := ch.profile.Region; r != "" {
		flags = append(flags, "--region", r)
		ch.clusterConfig.Region = r
	}

	prefix := helper.TrimNameByLength(clusterName, constants.MaxRolePrefixLength)
	flags = append(flags, "--operator-roles-prefix", prefix)
	ch.clusterConfig.Sts = true
	ch.clusterConfig.Aws = &ClusterConfigure.AWS{Sts: ClusterConfigure.Sts{OperatorRolesPrefix: prefix}}

	if ch.profile.ClusterConfig.OIDCConfig != "" {
		oidcConfigPrefix := helper.TrimNameByLength(clusterName, constants.MaxOIDCConfigPrefixLength)
		log.Logger.Infof("Preparing %s OIDC config with prefix %s for Platform API",
			ch.profile.ClusterConfig.OIDCConfig, oidcConfigPrefix)
		oidcConfigID, err := ch.resourcesHandler.PrepareOIDCConfig(
			ch.profile.ClusterConfig.OIDCConfig, "", oidcConfigPrefix,
		)
		if err != nil {
			return flags, err
		}
		flags = append(flags, "--oidc-config-id", oidcConfigID)
		ch.clusterConfig.Aws.Sts.OidcConfigID = oidcConfigID

		if !ch.profile.ClusterConfig.ManualCreationMode {
			err = ch.resourcesHandler.PrepareOperatorRolesByOIDCConfig(
				prefix, oidcConfigID, "", "", "", true, ch.profile.ChannelGroup,
			)
			if err != nil {
				return flags, err
			}
		}
	}

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
	if pc.VolumeSize != 0 {
		ch.clusterConfig.WorkerDiskSize = fmt.Sprintf("%dGiB", pc.VolumeSize)
	}

	log.Logger.Info("✅ V2 Add Networking defaults.")
	// V2 always set networking defaults
	networking := &ClusterConfigure.Networking{
		MachineCIDR: "10.0.0.0/16",
		PodCIDR:     "10.128.0.0/14",
		ServiceCIDR: "172.31.0.0/24",
		HostPrefix:  "23",
	}
	flags = append(flags,
		"--machine-cidr", networking.MachineCIDR, // Placeholder, it should be vpc CIDR
		"--service-cidr", networking.ServiceCIDR,
		"--pod-cidr", networking.PodCIDR,
		"--host-prefix", networking.HostPrefix,
	)
	ch.clusterConfig.Networking = networking
	log.Logger.Info("✅ V2 Add Networking.Type")
	ch.clusterConfig.Networking.Type = "OVNKubernetes"

	return flags, ch.saveToFile()
}

func resolvePlatformAPIVersion(profileVersion string) (string, bool) {
	if v := os.Getenv("HYPERFLEET_VERSION"); v != "" {
		return v, true
	}

	if profileVersion != "" && constants.VersionRawPattern.MatchString(profileVersion) {
		return profileVersion, true
	}

	if profileVersion != "" {
		log.Logger.Infof(
			"Skipping OCM version lookup for %q; Platform API resolves default (set HYPERFLEET_VERSION to pin)",
			profileVersion,
		)
	}
	return "", false
}

func prepareBYOVPCSubnets(
	rh *resourcesHandler, name string, profile *Profile, cfg *ClusterConfigure.ClusterConfig,
) (string, error) {
	cidr := constants.DefaultVPCCIDRValue
	if profile.ClusterConfig.NetworkingSet {
		cidr = cfg.Networking.MachineCIDR
	}

	vpcName := helper.TrimNameByLength(name, 20)
	if _, err := rh.PrepareVPC(vpcName, cidr, false, profile.ClusterConfig.SharedVPC); err != nil {
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

func (ch *clusterHandler) waitForHyperfleetClusterReady(timeoutMin int) error {
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
		// Normalize to lowercase for case-insensitive comparison (V2 API returns lowercase)
		phaseLower := strings.ToLower(phase)
		switch phaseLower {
		case strings.ToLower(string(v1alpha1.ClusterPhaseReady)):
			log.Logger.Infof("Cluster %s is ready now.", clusterKey)
			if err := ch.createHyperfleetNodePoolsAfterReady(); err != nil {
				return err
			}
			ch.recordHyperfleetClusterVersion(clusterKey)
			return nil
		case strings.ToLower(string(v1alpha1.ClusterPhaseDeleting)):
			return fmt.Errorf("cluster %s is %s now. Cannot wait for it ready", clusterKey, phase)
		case strings.ToLower(string(v1alpha1.ClusterPhaseWaitingForPlacement)), strings.ToLower(string(v1alpha1.ClusterPhaseProvisioning)), "":
			log.Logger.Infof("Cluster %s phase is %q, waiting for Ready", clusterKey, phase)
			time.Sleep(2 * time.Minute)
		default:
			return fmt.Errorf("unknown cluster phase %q", phase)
		}
	}

	return fmt.Errorf("timeout for cluster ready waiting after %d mins", timeoutMin)
}

func hyperfleetNodePoolReplicas(pc *ClusterConfig, poolCount int) (int, error) {
	if pc.Autoscale {
		return 0, errors.New("HyperFleet e2e setup does not support autoscaled default node pools")
	}
	n := 2
	if poolCount > 1 {
		n = 1
	}
	if pc.WorkerPoolReplicas != 0 {
		n = pc.WorkerPoolReplicas
	}
	return n, nil
}

func (ch *clusterHandler) createHyperfleetNodePoolsAfterReady() error {
	clusterKey := ch.clusterDetail.ClusterID
	if clusterKey == "" {
		clusterKey = ch.clusterDetail.ClusterName
	}
	if clusterKey == "" {
		return errors.New("missing cluster key for node pool creation")
	}

	pc := ch.profile.ClusterConfig
	instanceType := pc.InstanceType
	if instanceType == "" {
		instanceType = "m5.xlarge"
	}
	if ch.clusterConfig.Subnets == nil {
		return fmt.Errorf("no private subnets in cluster-config; cannot create node pools")
	}
	subnets := helper.ParseCommaSeparatedStrings(ch.clusterConfig.Subnets.PrivateSubnetIds)
	if len(subnets) == 0 {
		return fmt.Errorf("no private subnets in cluster-config; cannot create node pools")
	}

	poolNames := []string{"workers"}
	if pc.MultiAZ && len(subnets) > 1 {
		poolNames = make([]string, 0, len(subnets))
		for i := range subnets {
			poolNames = append(poolNames, fmt.Sprintf("workers-%d", i))
		}
	}
	replicas, err := hyperfleetNodePoolReplicas(pc, len(poolNames))
	if err != nil {
		return err
	}

	mp := ch.rosaClient.MachinePool
	for i, name := range poolNames {
		subnet := subnets[i]
		flags := []string{
			"--replicas", fmt.Sprintf("%d", replicas),
			"--instance-type", instanceType,
			"--subnet", subnet,
			"-y",
		}
		if ch.clusterConfig.WorkerDiskSize != "" {
			flags = append(flags, "--disk-size", ch.clusterConfig.WorkerDiskSize)
		}
		log.Logger.Infof("Creating Hyperfleet node pool %s on subnet %s", name, subnet)
		if _, err := mp.CreateMachinePool(clusterKey, name, flags...); err != nil {
			if strings.Contains(err.Error(), "already exists") {
				log.Logger.Infof("Hyperfleet node pool %s already exists, continuing", name)
				continue
			}
			return fmt.Errorf("create node pool %q: %w", name, err)
		}
	}

	return ch.waitForHyperfleetNodePools(clusterKey, poolNames)
}

func (ch *clusterHandler) waitForHyperfleetNodePools(clusterKey string, poolNames []string) error {
	end := time.Now().Add(20 * time.Minute)
	var lastErr error
	for time.Now().Before(end) {
		lastErr = nil
		allReady := true
		for _, name := range poolNames {
			out, err := ch.rosaClient.MachinePool.DescribeMachinePool(clusterKey, name)
			if err != nil {
				lastErr = err
				allReady = false
				break
			}
			nodePool, err := ch.rosaClient.MachinePool.ReflectNodePoolDescription(out)
			if err != nil {
				lastErr = err
				allReady = false
				break
			}
			if nodePool.State != string(v1alpha1.NodePoolPhaseReady) {
				allReady = false
				break
			}
		}
		if allReady {
			log.Logger.Infof("Hyperfleet node pools ready: %d", len(poolNames))
			return nil
		}
		time.Sleep(30 * time.Second)
	}
	if lastErr != nil {
		return fmt.Errorf("timeout waiting for %d node pool(s) on cluster %s: %w", len(poolNames), clusterKey, lastErr)
	}
	return fmt.Errorf("timeout waiting for %d node pool(s) on cluster %s", len(poolNames), clusterKey)
}

func (ch *clusterHandler) recordHyperfleetClusterVersion(clusterKey string) {
	v, err := ch.rosaClient.Cluster.GetClusterVersion(clusterKey)
	if err != nil {
		log.Logger.Warnf("Could not read cluster version for cluster-config: %v", err)
		return
	}
	if v.RawID == "" {
		return
	}
	ch.clusterConfig.Version = &v
	if ch.profile.ChannelGroup != "" {
		ch.clusterConfig.Version.ChannelGroup = ch.profile.ChannelGroup
	}
	if ch.profile.Version != "" {
		ch.clusterConfig.Version.VersionRequirement = ch.profile.Version
	}
	_ = ch.saveToFile()
}
