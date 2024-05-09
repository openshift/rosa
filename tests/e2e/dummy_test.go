package e2e

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	TC "github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/log"
	"github.com/openshift/rosa/tests/utils/profilehandler"
	PH "github.com/openshift/rosa/tests/utils/profilehandler"
)

var _ = Describe("ROSA CLI Test", func() {
	Describe("Dummy test", func() {
		It("Dummy", func() {
			str := "dummy string"
			Expect(str).ToNot(BeEmpty())
			log.Logger.Infof("This is a dummy test to check everything is fine by executing jobs. Please remove me once other tests are added")
		})
	})
	Describe("Profile test", func() {
		It("ProfileParserTest", func() {
			profile := PH.LoadProfileYamlFileByENV()
			log.Logger.Infof("Got configured profile prefix: %v", *profile)
			log.Logger.Infof("Got configured profile: %v", profile.NamePrefix)
			log.Logger.Infof("Got configured cluster profile: %v", *profile.ClusterConfig)
			log.Logger.Infof("Got configured account role profile: %v", *profile.AccountRoleConfig)
		})
		It("TestENVSetup", func() {
			log.Logger.Infof("Got dir of out: %v", TC.Test.OutputDir)
		})
		It("TestPrepareClusterByProfile", func() {
			client := rosacli.NewClient()
			profile := PH.LoadProfileYamlFileByENV()
			cluster, err := PH.CreateClusterByProfile(profile, client, true)
			Expect(err).ToNot(HaveOccurred())
			fmt.Println(cluster.ID)
		})
		It("TestRemovingFunc", func() {
			s := strings.Split("", ",")
			s = common.RemoveFromStringSlice(s, "")
			fmt.Println(len(s))
		})
	})
	Describe("ocm-common test", func() {
		It("VPCClientTesting", func() {
			vpcClient, err := profilehandler.PrepareVPC("us-east-1", "xueli-test", "10.0.0.0/16")
			Expect(err).ToNot(HaveOccurred())
			defer vpcClient.DeleteVPCChain(true)
			subnets, err := profilehandler.PrepareSubnets(vpcClient, "us-east-1", []string{}, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(subnets)).To(Equal(2))
		})

	})
	Describe("ROSAClientServiceTestingCode testing", func() {
		var rosaClient *rosacli.Client
		var clusterID string
		BeforeEach(func() {
			rosaClient = rosacli.NewClient()
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(BeEmpty())
		})
		It("IngressServiceTesting", func() {
			output, err := rosaClient.Ingress.ListIngress(clusterID)
			Expect(err).ToNot(HaveOccurred())
			ingressList, err := rosaClient.Ingress.ReflectIngressList(output)
			Expect(err).ToNot(HaveOccurred())
			Expect(ingressList.Ingresses).ToNot(BeEmpty())
			Expect(ingressList.Ingresses[0].LBType).ToNot(BeEmpty())
		})
	})
	Describe("logstreamtest", func() {
		It("", func() {
			funcA := func(causeError bool) error {
				rosacli.NewClient().OCMResource.ListRegion()
				log.Logger.Debugf("I am debug message with caseuError %v", causeError)
				if causeError {
					return fmt.Errorf("test")
				}
				return nil
			}
			// Expect(funcA(true)).ToNot(HaveOccurred())
			Expect(funcA(false)).ToNot(HaveOccurred())
		})
	})
})
