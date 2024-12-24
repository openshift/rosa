package e2e

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ciConfig "github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/occli"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/handler"
	"github.com/openshift/rosa/tests/utils/helper"
	. "github.com/openshift/rosa/tests/utils/log"
)

var _ = Describe("ROSA CLI Test", func() {
	Describe("Dummy test", func() {
		It("Dummy", func() {
			str := "dummy string"
			Expect(str).ToNot(BeEmpty())
			Logger.Infof("This is a dummy test to check everything is fine by executing jobs. " +
				"Please remove me once other tests are added")
		})
	})
	Describe("Profile test", func() {
		It("ProfileParserTest", func() {
			profile := handler.LoadProfileYamlFileByENV()
			Logger.Infof("Got configured profile: %v", *profile)
			Logger.Infof("Got configured profile prefix: %v", profile.NamePrefix)
			Logger.Infof("Got configured cluster profile: %v", *profile.ClusterConfig)
			Logger.Infof("Got configured account role profile: %v", *profile.AccountRoleConfig)
		})
		It("TestENVSetup", func() {
			Logger.Infof("Got dir of out: %v", ciConfig.Test.OutputDir)
		})
		It("TestPrepareClusterByProfile", func() {
			client := rosacli.NewClient()
			profile := handler.LoadProfileYamlFileByENV()
			clusterHandler, err := handler.NewTempClusterHandler(client, profile)
			Expect(err).ToNot(HaveOccurred())
			err = clusterHandler.CreateCluster(true)
			defer clusterHandler.Destroy()
			Expect(err).ToNot(HaveOccurred())
			fmt.Println(clusterHandler.GetClusterDetail().ClusterID)
		})
		It("TestRemovingFunc", func() {
			s := strings.Split("", ",")
			s = helper.RemoveFromStringSlice(s, "")
			fmt.Println(len(s))
		})
	})
	Describe("ocm-common test", func() {
		It("VPCClientTesting", func() {
			client := rosacli.NewClient()
			region := "us-east-1"
			resourcesHandler, err := handler.NewTempResourcesHandler(client, region, "", "")
			Expect(err).ToNot(HaveOccurred())
			vpcClient, err := resourcesHandler.PrepareVPC("xueli-test", "10.0.0.0/16", false, false)
			Expect(err).ToNot(HaveOccurred())
			defer resourcesHandler.DestroyResources()
			subnets, err := resourcesHandler.PrepareSubnets([]string{}, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(subnets)).To(Equal(2))
			_, ip, ca, err := vpcClient.LaunchProxyInstance("us-east-1a", "xueli-test", ciConfig.Test.OutputDir)

			Expect(err).ToNot(HaveOccurred())
			fmt.Println(ip)
			fmt.Println(ca)
			Logger.Infof("Got configured proxy ip: %v", ip)
			Logger.Infof("Got configured proxy ca: %v", ca)
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
				Logger.Debugf("I am debug message with caseuError %v", causeError)
				if causeError {
					return fmt.Errorf("test")
				}
				return nil
			}
			// Expect(funcA(true)).ToNot(HaveOccurred())
			Expect(funcA(false)).ToNot(HaveOccurred())
		})
	})
	// Used to check the sensitive data filterting
	Describe("logstreamsensitivedata", func() {
		It("", func() {
			Logger.Info("rosa create cluster --billing-account 012345678912")
			Logger.Info("--password antyhingoutofordinary endofpassword")
			Logger.Info("beginning --client-id thisisclient end")
			Expect(false).To(BeTrue())
		})
	})
})

var _ = Describe("OC CLI Test", func() {
	Describe("Test created kubeconfig", func() {
		It("Test", func() {
			ocClient, err := occli.NewOCClient()
			Expect(err).ShouldNot(HaveOccurred())
			stdout, err := ocClient.Run("oc project dedicated-admin", 3)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Now using project \"dedicated-admin\""))
		})
	})
})
