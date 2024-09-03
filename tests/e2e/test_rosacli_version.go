package e2e

import (
	"github.com/Masterminds/semver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Get CLI version",
	labels.Feature.Version,
	func() {
		defer GinkgoRecover()

		var (
			rosaClient *rosacli.Client
		)

		BeforeEach(func() {
			By("Init the client")
			rosaClient = rosacli.NewClient()
			configFile, err := common.CreateTempOCMConfig()
			Expect(err).ToNot(HaveOccurred())
			rosaClient.Runner.AddEnvVar("OCM_CONFIG", configFile)
		})

		It("can get the version of rosa CLI while logged out - [id:73743]",
			labels.Medium, labels.Runtime.OCMResources,
			func() {
				By("Make sure the CLI is logged out")
				buf, err := rosaClient.Runner.Cmd("whoami").Run()
				stdout := rosaClient.Parser.TextData.Input(buf).Parse().Tip()
				Expect(stdout).To(ContainSubstring("Not logged in"))
				Expect(err).To(HaveOccurred())

				By("Get the version output")
				buf, err = rosaClient.Runner.Cmd("version").Run()
				Expect(err).To(BeNil())
				stdout = rosaClient.Parser.TextData.Input(buf).Parse().Output()
				By("Check the version output")
				Expect(stdout).NotTo(ContainSubstring("Not logged in"))
				Expect(stdout).To(ContainSubstring(info.DefaultVersion))

				By("Get the client version output")
				buf, err = rosaClient.Runner.Cmd("version", "--client").Run()
				Expect(err).To(BeNil())
				stdout = rosaClient.Parser.TextData.Input(buf).Parse().Output()
				By("Check the client version output")
				Expect(stdout).NotTo(ContainSubstring("Not logged in"))
				Expect(stdout).To(ContainSubstring(info.DefaultVersion))
			},
		)

		It("list versions can work correctly for hosted-cp cluster via ROSA cli - [id:62088]",
			labels.High,
			func() {
				By("Init the client")
				rosaClient = rosacli.NewClient()

				By("list the versions with --hosted-cp")
				versionService := rosaClient.Version

				output, err := versionService.ListVersions("", true)
				Expect(err).ToNot(HaveOccurred())
				Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					Should(ContainSubstring(
						"INFO: Hosted cluster upgrades are cluster-based. " +
							"To list available upgrades for a cluster, please use 'rosa list upgrades'"))

				By("Get the default version")
				vList, err := versionService.ListAndReflectVersions("", true)
				Expect(err).ToNot(HaveOccurred())
				defaultVersion := vList.DefaultVersion().Version

				By("list the versions with channel group")
				channelGroups := []string{"stable", "", "candidate", "nightly"}
				for _, c := range channelGroups {
					verList, err := versionService.ListAndReflectJsonVersions(c, true)
					Expect(err).ToNot(HaveOccurred())
					for _, v := range verList {
						if !v.Enabled {
							continue
						}
						Expect(v.HCPEnabled).To(BeTrue())
						if c == "" {
							Expect(v.ChannelGroup).To(Equal("stable"))
						} else {
							Expect(v.ChannelGroup).To(Equal(c))
						}
						if v.Default {
							Expect(defaultVersion).To(Equal(v.RAWID))
						}
						baseVersionSemVer, err := semver.NewVersion(v.RAWID)
						Expect(err).ToNot(HaveOccurred())
						if baseVersionSemVer.Major() == 4 {
							Expect(baseVersionSemVer.Minor()).To(BeNumerically(">=", 13))
						}
					}
				}

			},
		)
	})
