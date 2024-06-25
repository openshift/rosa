package e2e

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	ph "github.com/openshift/rosa/tests/utils/profilehandler"
)

const YES = "yes"

var _ = Describe("Edit default ingress",
	labels.Feature.Ingress,
	func() {
		defer GinkgoRecover()

		var (
			clusterID      string
			profile        ph.Profile
			rosaClient     *rosacli.Client
			ingressService rosacli.IngressService
			isHosted       bool
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			ingressService = rosaClient.Ingress

			By("Check cluster is hosted")
			var err error
			isHosted, err = rosaClient.Cluster.IsHostedCPCluster(clusterID)
			Expect(err).ToNot(HaveOccurred())

			By("Load the profile")
			profile = *ph.LoadProfileYamlFileByENV()
		})

		It("can update on rosa HCP cluster - [id:63323]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				By("Skip testing if the cluster is not a HCP cluster")
				if !isHosted {
					SkipNotHosted()
				}

				By("Retrieve cluster and get default ingress id")
				output, err := ingressService.ListIngress(clusterID)
				Expect(err).ToNot(HaveOccurred())

				ingressList, err := ingressService.ReflectIngressList(output)
				Expect(err).ToNot(HaveOccurred())
				var defaultID, originalValue string
				for _, v := range ingressList.Ingresses {
					if v.Default == YES {
						defaultID = v.ID
						originalValue = v.Private
					}
				}

				By("Edit the default ingress on rosa HCP cluster to different value")
				updatedValue := "no"
				if originalValue == "no" {
					updatedValue = YES
				}
				testvalue := map[string]string{
					YES:  "true",
					"no": "false",
				}
				cmdFlag := fmt.Sprintf("--private=%s", testvalue[updatedValue])
				output, err = ingressService.EditIngress(clusterID, defaultID,
					cmdFlag)
				Expect(err).ToNot(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"INFO: Updated ingress '%s' on cluster '%s'",
						defaultID,
						clusterID))

				defer func() {
					_, err = ingressService.EditIngress(clusterID, defaultID,
						fmt.Sprintf("--private=%s", testvalue[originalValue]))
					Expect(err).ToNot(HaveOccurred())

					output, err = ingressService.ListIngress(clusterID)
					Expect(err).ToNot(HaveOccurred())

					ingressList, err = ingressService.ReflectIngressList(output)
					Expect(err).ToNot(HaveOccurred())

					in := ingressList.Ingress(defaultID)
					Expect(in.Private).To(Equal(originalValue))
				}()

				output, err = ingressService.ListIngress(clusterID)
				Expect(err).ToNot(HaveOccurred())

				ingressList, err = ingressService.ReflectIngressList(output)
				Expect(err).ToNot(HaveOccurred())

				in := ingressList.Ingress(defaultID)
				Expect(in.Private).To(Equal(updatedValue))

				By("Edit the default ingress on rosa HCP cluster with current value")
				output, err = ingressService.EditIngress(clusterID, defaultID, cmdFlag)
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"WARN: No need to update ingress as there are no changes"))

				By("Edit the default ingress only with --private")
				output, err = ingressService.EditIngress(clusterID, defaultID, "--private")
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				if updatedValue == YES {
					Expect(textData).
						Should(ContainSubstring(
							"WARN: No need to update ingress as there are no changes"))
				} else {
					Expect(textData).
						Should(ContainSubstring(
							"Updated ingress '%s' on cluster '%s'", defaultID, clusterID))
				}

				By("Run command to edit a default ingress with --label-match")
				output, err = ingressService.EditIngress(clusterID, defaultID,
					"--label-match", "aaa=bbb,ccc=ddd")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"ERR: Updating route selectors is not supported for Hosted Control Plane clusters"))
			})

		It("change load balancer type - [id:64767]",
			labels.Critical,
			labels.Runtime.Day2,
			func() {
				By("Skip testing if the cluster is not a Classic cluster")
				if isHosted {
					SkipNotClassic()
				}

				output, err := ingressService.ListIngress(clusterID)
				Expect(err).ToNot(HaveOccurred())
				ingressList, err := ingressService.ReflectIngressList(output)
				Expect(err).ToNot(HaveOccurred())

				defaultIngress := func(ingressList rosacli.IngressList) (*rosacli.Ingress, bool) {
					for _, ingress := range ingressList.Ingresses {
						if ingress.Default == YES {
							return ingress, true
						}
					}
					return nil, false
				}
				ingress, exists := defaultIngress(*ingressList)
				Expect(exists).To(BeTrue())
				defaultID := ingress.ID
				Expect(defaultID).ToNot(BeNil())
				updatingIngresType := "nlb"
				if ingress.LBType == "nlb" {
					updatingIngresType = "classic"
				}
				output, err = ingressService.EditIngress(clusterID, defaultID, "--lb-type", updatingIngresType)
				Expect(err).ToNot(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Updated ingress '%s'", defaultID))

				defer ingressService.EditIngress(clusterID, defaultID, "--lb-type", ingress.LBType)

				output, err = ingressService.ListIngress(clusterID)
				Expect(err).ToNot(HaveOccurred())
				ingressList, err = ingressService.ReflectIngressList(output)
				Expect(err).ToNot(HaveOccurred())
				updatedIngress, _ := defaultIngress(*ingressList)
				Expect(updatedIngress.LBType).Should(Equal(updatingIngresType))

				output, err = ingressService.EditIngress(clusterID, defaultID, "--lb-type", ingress.LBType)
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Updated ingress '%s'", defaultID))

				output, err = ingressService.ListIngress(clusterID)
				Expect(err).ToNot(HaveOccurred())
				ingressList, err = ingressService.ReflectIngressList(output)
				Expect(err).ToNot(HaveOccurred())
				ingress, _ = defaultIngress(*ingressList)
				Expect(ingress.LBType).Should(ContainSubstring(ingress.LBType))
			})
		It("can update ingress controller attributes - [id:65799]",
			labels.Critical,
			labels.Runtime.Day2,
			func() {
				By("Skip testing if the cluster is not a Classic cluster")
				if isHosted {
					SkipNotClassic()
				}

				output, err := ingressService.ListIngress(clusterID)
				Expect(err).ToNot(HaveOccurred())

				ingressList, err := ingressService.ReflectIngressList(output)
				Expect(err).ToNot(HaveOccurred())
				defaultIngress := func(ingressList rosacli.IngressList) (*rosacli.Ingress, bool) {
					for _, ingress := range ingressList.Ingresses {
						if ingress.Default == YES {
							return ingress, true
						}
					}
					return nil, false
				}

				ingress, exists := defaultIngress(*ingressList)
				Expect(exists).To(BeTrue())
				defaultID := ingress.ID

				// Recover the ingress
				defer func() {
					flags := []string{"--excluded-namespaces", ingress.ExcludeNamespace,
						"--route-selector", common.ReplaceCommaSpaceWithComma(ingress.RouteSelectors),
						"--namespace-ownership-policy", ingress.NamespaceOwnershipPolicy,
						"--wildcard-policy", ingress.WildcardPolicy,
					}
					ingressService.EditIngress(clusterID, defaultID, flags...)
				}()
				updatingRouteSelector := "app-65799=test-65799-2,app2=test-65799"
				output, err = ingressService.EditIngress(
					clusterID,
					defaultID,
					"--excluded-namespaces", "test-ns1,test-ns2",
					"--route-selector", updatingRouteSelector,
					"--namespace-ownership-policy", "Strict",
					"--wildcard-policy", "WildcardsDisallowed",
				)
				Expect(err).ToNot(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Updated ingress '%s'", defaultID))

				output, err = ingressService.ListIngress(clusterID)
				Expect(err).ToNot(HaveOccurred())

				ingressList, err = ingressService.ReflectIngressList(output)
				Expect(err).ToNot(HaveOccurred())

				ingress, _ = defaultIngress(*ingressList)
				Expect(ingress.ExcludeNamespace).Should(ContainSubstring("test-ns1"))
				Expect(ingress.ExcludeNamespace).Should(ContainSubstring("test-ns2"))
				Expect(ingress.RouteSelectors).Should(ContainSubstring("app-65799=test-65799"))
				Expect(ingress.RouteSelectors).Should(ContainSubstring("app2=test-65799"))
				Expect(ingress.NamespaceOwnershipPolicy).Should(ContainSubstring("Strict"))
				Expect(ingress.WildcardPolicy).Should(ContainSubstring("WildcardsDisallowed"))
			})
		It("can change labels and private - [id:38835]",
			labels.Critical,
			labels.Runtime.Day2,
			func() {
				By("Skip testing if the cluster is not a Classic cluster")
				if isHosted {
					SkipNotClassic()
				}

				By("Record ingress default value")
				output, err := rosaClient.Ingress.ListIngress(clusterID)
				Expect(err).ToNot(HaveOccurred())
				ingressList, err := rosaClient.Ingress.ReflectIngressList(output)
				Expect(err).ToNot(HaveOccurred())
				defaultIngress := ingressList.Ingresses[0]
				originalPrivate := defaultIngress.Private == YES
				originalRouteSelectors := defaultIngress.RouteSelectors

				By("Check edit ingress help message")
				output, err = rosaClient.Ingress.EditIngress(clusterID, "-h")
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("--label-match string"))

				By("Edit ingress with --label-match and --private")
				labelMatch := "label-38835=label-value-38835,label-38835-2=label-value-38835-2"
				output, err = rosaClient.Ingress.EditIngress(clusterID,
					"apps",
					"--label-match", labelMatch,
					fmt.Sprintf("--private=%v", !originalPrivate),
					"-y",
				)
				if profile.ClusterConfig.PrivateLink && !isHosted {
					Expect(err).To(HaveOccurred())
					Expect(output.String()).Should(ContainSubstring("Can't update listening mode on an AWS Private Link cluster"))

					By("Edit label-match only")
					output, err = rosaClient.Ingress.EditIngress(clusterID,
						"apps",
						"--label-match", labelMatch,
						"-y",
					)
					Expect(err).ToNot(HaveOccurred())
					defer rosaClient.Ingress.EditIngress(clusterID,
						"apps",
						"--label-match", common.ReplaceCommaSpaceWithComma(originalRouteSelectors),
						"-y",
					)

					By("Describe ingress and check")
					ingressDescription, err := rosaClient.Ingress.DescribeIngressAndReflect(clusterID, "apps")
					Expect(err).ToNot(HaveOccurred())
					// Below is workaround due to known issue
					ingressRouteSelectors := strings.Split(ingressDescription.RouteSelectors, " ")
					for index, ingressRS := range ingressRouteSelectors {
						wgString := strings.TrimSuffix(strings.TrimPrefix(ingressRS, "map["), "]")
						wgString = strings.ReplaceAll(wgString, ":", "=")
						ingressRouteSelectors[index] = wgString
					}
					// Workaround finished
					expectedRouteSelectors := common.ParseCommaSeparatedStrings(labelMatch)

					Expect(len(ingressRouteSelectors)).To(Equal(len(expectedRouteSelectors)))

					for _, expectLabel := range expectedRouteSelectors {
						Expect(expectLabel).To(BeElementOf(ingressRouteSelectors))
					}

					return
				}
				Expect(err).ToNot(HaveOccurred())
				defer rosaClient.Ingress.EditIngress(clusterID,
					"apps",
					"--label-match", common.ReplaceCommaSpaceWithComma(originalRouteSelectors),
					fmt.Sprintf("--private=%v", originalPrivate),
					"-y",
				)

				By("List ingress to check")
				output, err = rosaClient.Ingress.ListIngress(clusterID)
				Expect(err).ToNot(HaveOccurred())
				ingressList, err = rosaClient.Ingress.ReflectIngressList(output)
				Expect(err).ToNot(HaveOccurred())

				defaultIngress = ingressList.Ingresses[0]
				Expect(defaultIngress.Private == YES).To(Equal(!originalPrivate))

				ingressRouteSelectors := common.ParseCommaSeparatedStrings(defaultIngress.RouteSelectors)
				expectedRouteSelectors := common.ParseCommaSeparatedStrings(labelMatch)

				Expect(len(ingressRouteSelectors)).To(Equal(len(expectedRouteSelectors)))

				for _, expectLabel := range expectedRouteSelectors {
					Expect(expectLabel).To(BeElementOf(ingressRouteSelectors))
				}
			})
	})
