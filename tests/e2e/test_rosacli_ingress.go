package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"

	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Edit default ingress",
	labels.Feature.Ingress,
	func() {
		defer GinkgoRecover()

		var (
			clusterID      string
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
					if v.Default == "yes" {
						defaultID = v.ID
						originalValue = v.Private
					}
				}

				By("Edit the default ingress on rosa HCP cluster to different value")
				updatedValue := "no"
				if originalValue == "no" {
					updatedValue = "yes"
				}
				testvalue := map[string]string{
					"yes": "true",
					"no":  "false",
				}
				cmdFlag := fmt.Sprintf("--private=%s", testvalue[updatedValue])
				output, err = ingressService.EditIngress(clusterID, defaultID,
					cmdFlag)
				Expect(err).ToNot(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("INFO: Updated ingress '%s' on cluster '%s'", defaultID, clusterID))

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
				Expect(textData).Should(ContainSubstring("WARN: No need to update ingress as there are no changes"))

				By("Edit the default ingress only with --private")
				output, err = ingressService.EditIngress(clusterID, defaultID, "--private")
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				if updatedValue == "yes" {
					Expect(textData).Should(ContainSubstring("WARN: No need to update ingress as there are no changes"))
				} else {
					Expect(textData).Should(ContainSubstring("Updated ingress '%s' on cluster '%s'", defaultID, clusterID))
				}

				By("Run command to edit a default ingress with --label-match")
				output, err = ingressService.EditIngress(clusterID, defaultID,
					"--label-match", "aaa=bbb,ccc=ddd")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("ERR: Updating route selectors is not supported for Hosted Control Plane clusters"))
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
						if ingress.Default == "yes" {
							return &ingress, true
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
						if ingress.Default == "yes" {
							return &ingress, true
						}
					}
					return nil, false
				}

				ingress, exists := defaultIngress(*ingressList)
				Expect(exists).To(BeTrue())
				defaultID := ingress.ID
				output, err = ingressService.EditIngress(clusterID, defaultID, "--excluded-namespaces", "test-ns1,test-ns2", "--route-selector",
					"app1=test1,app2=test2", "--namespace-ownership-policy", "Strict", "--wildcard-policy", "WildcardsDisallowed")
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
				Expect(ingress.RouteSelectors).Should(ContainSubstring("app1=test1"))
				Expect(ingress.RouteSelectors).Should(ContainSubstring("app2=test2"))
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
				originalPrivate := defaultIngress.Private == "yes"
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
				Expect(defaultIngress.Private == "yes").To(Equal(!originalPrivate))

				ingressRouteSelectors := common.ParseCommaSeparatedStrings(defaultIngress.RouteSelectors)
				expectedRouteSelectors := common.ParseCommaSeparatedStrings(labelMatch)

				Expect(len(ingressRouteSelectors)).To(Equal(len(expectedRouteSelectors)))

				for _, expectLabel := range expectedRouteSelectors {
					Expect(expectLabel).To(BeElementOf(ingressRouteSelectors))
				}
			})
		It("can update ingress components (oauth, downloads, console) - [id:72868]",
			labels.Medium,
			labels.Runtime.Day2,
			func() {

				By("Record ingress default value")
				output, err := rosaClient.Ingress.ListIngress(clusterID)
				Expect(err).ToNot(HaveOccurred())
				ingressList, err := rosaClient.Ingress.ReflectIngressList(output)
				Expect(err).ToNot(HaveOccurred())
				defaultIngress := ingressList.Ingresses[0]

				By("Check edit ingress help message")
				output, err = rosaClient.Ingress.EditIngress(clusterID, "-h")
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("--component-routes"))

				By("Edit ingress with --component-routes")
				componentRoutes := "oauth: hostname=oauth.hostname.com;tlsSecretRef=oauth-secret,downloads: hostname=downloads.hostname.com;tlsSecretRef=downloads-secret,console: hostname=console.hostname.com;tlsSecretRef=console-secret"
				output, err = rosaClient.Ingress.EditIngress(clusterID,
					defaultIngress.ID,
					"--component-routes", componentRoutes,
				)
				Expect(err).ToNot(HaveOccurred())
				defer rosaClient.Ingress.EditIngress(clusterID,
					defaultIngress.ID,
					"--component-routes", "oauth: hostname=oauth.hostname.com;tlsSecretRef=oauth-secret,downloads: hostname=downloads.hostname.com;tlsSecretRef=downloads-secret,console: hostname=console.hostname.com;tlsSecretRef=console-secret",
				)

				By("List ingress to check")
				output, err = rosaClient.Ingress.ListIngress(clusterID)
				Expect(err).ToNot(HaveOccurred())
			})
		It("cannot update ingress components with incorrect syntax - [id:72868]",
			labels.Medium,
			labels.Runtime.Day2,
			func() {
				By("Record ingress default value")
				output, err := rosaClient.Ingress.ListIngress(clusterID)
				Expect(err).ToNot(HaveOccurred())
				ingressList, err := rosaClient.Ingress.ReflectIngressList(output)
				Expect(err).ToNot(HaveOccurred())
				defaultIngress := ingressList.Ingresses[0]

				By("Edit ingress with --component-routes")
				componentRoutes := "oauth: hostname:custom1;tlsSecretRef=custom1,downloads: hostname=custom2;tlsSecretRef=custom2,console: hostname=custom3;tlsSecretRef=custom3"
				output, err = rosaClient.Ingress.EditIngress(clusterID,
					defaultIngress.ID,
					"--component-routes", componentRoutes,
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("An error occurred whilst parsing the supplied component routes: only the name of the component should be followed by ':'"))
				By("List ingress to check")
				output, err = rosaClient.Ingress.ListIngress(clusterID)
				Expect(err).ToNot(HaveOccurred())
			})
		It("cannot update ingress components with incorrect number of components - [id:72868]",
			labels.Medium,
			labels.Runtime.Day2,
			func() {
				By("Record ingress default value")
				output, err := rosaClient.Ingress.ListIngress(clusterID)
				Expect(err).ToNot(HaveOccurred())
				ingressList, err := rosaClient.Ingress.ReflectIngressList(output)
				Expect(err).ToNot(HaveOccurred())
				defaultIngress := ingressList.Ingresses[0]

				By("Edit ingress with --component-routes")
				componentRoutes := "oauth: hostname=custom1;tlsSecretRef=custom1"
				output, err = rosaClient.Ingress.EditIngress(clusterID,
					defaultIngress.ID,
					"--component-routes", componentRoutes,
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("An error occurred whilst parsing the supplied component routes: the expected amount of component routes is 3, but 1 have been supplied"))
				By("List ingress to check")
				_, err = rosaClient.Ingress.ListIngress(clusterID)
				Expect(err).ToNot(HaveOccurred())
			})
	})
