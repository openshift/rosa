package e2e

import (
	"github.com/Masterminds/semver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/helper"
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
		})

		It("can get the version of rosa CLI while logged out - [id:73743]",
			labels.Medium, labels.Runtime.OCMResources,
			func() {
				By("Make sure the CLI is logged out")
				configFile, err := helper.CreateTempOCMConfig()
				Expect(err).ToNot(HaveOccurred())
				rosaClient.Runner.AddEnvVar("OCM_CONFIG", configFile)
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
			labels.High, labels.Runtime.OCMResources,
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

		It("list versions can work correctly via ROSA cli - [id:38810]",
			labels.High, labels.Runtime.OCMResources,
			func() {

				const STABLE_CHANNEL = "stable"
				const CANDIDATE_CHANNEL = "candidate"
				versionService := rosaClient.Version

				By("Display the version help page")
				output, err := versionService.ListVersions("", false, "-h")
				Expect(err).ToNot(HaveOccurred())
				stdout := rosaClient.Parser.TextData.Input(output).Parse().Output()

				By("Check the output of the help page")
				Expect(stdout).To(ContainSubstring("rosa list versions [flags]"))
				Expect(stdout).To(ContainSubstring("versions, version"))
				Expect(stdout).To(ContainSubstring("--channel-group string"))

				By("Display the version on the stable channel")
				rosaClient.Runner.UnsetArgs()
				output, err = versionService.ListVersions(STABLE_CHANNEL, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("DEPRECATED: Available upgrades in 'rosa list versions' are deprecated"))
				Expect(output.String()).To(ContainSubstring("please use 'rosa list upgrades --cluster <cluster_id>'"))
				stdout = rosaClient.Parser.TextData.Input(output).Parse().Output()

				By("Check the output of the stable versions")
				Expect(stdout).To(ContainSubstring("AVAILABLE UPGRADES"))
				verList, err := versionService.ListAndReflectJsonVersions(STABLE_CHANNEL, false)
				Expect(err).ToNot(HaveOccurred())
				for _, v := range verList {
					Expect(v.ChannelGroup).To(Equal(STABLE_CHANNEL))
					baseVersionSemVer, err := semver.NewVersion(v.RAWID)
					Expect(err).ToNot(HaveOccurred())
					if baseVersionSemVer.Major() == 4 {
						Expect(baseVersionSemVer.Minor()).To(BeNumerically(">=", 7))
					}
				}

				By("Display the version on the candidate channel")
				rosaClient.Runner.UnsetArgs()
				output, err = versionService.ListVersions(CANDIDATE_CHANNEL, false)
				Expect(err).ToNot(HaveOccurred())
				stdout = rosaClient.Parser.TextData.Input(output).Parse().Output()

				By("Check the output of the candidate versions")
				Expect(stdout).To(ContainSubstring("AVAILABLE UPGRADES"))
				verList, err = versionService.ListAndReflectJsonVersions(CANDIDATE_CHANNEL, false)
				Expect(err).ToNot(HaveOccurred())
				for _, v := range verList {
					Expect(v.ChannelGroup).To(Equal(CANDIDATE_CHANNEL))
					baseVersionSemVer, err := semver.NewVersion(v.RAWID)
					Expect(err).ToNot(HaveOccurred())
					if baseVersionSemVer.Major() == 4 {
						Expect(baseVersionSemVer.Minor()).To(BeNumerically(">=", 7))
					}
				}

				By("Display the version on the stable channel with the debug flag")
				rosaClient.Runner.UnsetArgs()
				output, err = versionService.ListVersions("", false, "--debug")
				Expect(err).ToNot(HaveOccurred())
				stdout = rosaClient.Parser.TextData.Input(output).Parse().Output()

				By("Check the output of the stable versions with the debug flag")
				Expect(stdout).To(ContainSubstring("level=debug"))
				Expect(stdout).To(ContainSubstring("AVAILABLE UPGRADES"))

				By("Display the version on the stable channel with an invalid flag")
				rosaClient.Runner.UnsetArgs()
				output, err = versionService.ListVersions("", false, "--debug", "--invalidflag")
				Expect(err).To(HaveOccurred())
				stdout = rosaClient.Parser.TextData.Input(output).Parse().Output()

				By("Check the output of the stable versions with an invalid flag")
				Expect(stdout).To(ContainSubstring("unknown flag"))
				Expect(stdout).To(ContainSubstring("rosa list versions [flags]"))
				Expect(stdout).To(ContainSubstring("versions, version"))
				Expect(stdout).To(ContainSubstring("--channel-group string"))

			},
		)

	})
