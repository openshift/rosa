package e2e

import (
	"errors"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/handler"
	"github.com/openshift/rosa/tests/utils/helper"
)

func validateIDPOutput(textData string, clusterID string, idpName string) {
	Expect(textData).Should(ContainSubstring("Configuring IDP for cluster '%s'", clusterID))
	Expect(textData).Should(ContainSubstring("Identity Provider '%s' has been created", idpName))
}

var _ = Describe("Edit IDP",
	labels.Feature.IDP,
	func() {
		defer GinkgoRecover()

		var (
			clusterID  string
			rosaClient *rosacli.Client
			idpService rosacli.IDPService
			profile    *handler.Profile
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the clients")
			rosaClient = rosacli.NewClient()
			idpService = rosaClient.IDP

			By("Load the profile")
			profile = handler.LoadProfileYamlFileByENV()

			if profile.ClusterConfig.AdminEnabled {
				// Delete the day1 created admin. DON'T user it in day1-post case
				idpService.DeleteIDP(clusterID, "cluster-admin")
			}
		})

		AfterEach(func() {
			By("Clean remaining resources")
			var errorList []error
			errorList = append(errorList, rosaClient.CleanResources(clusterID))
			Expect(errors.Join(errorList...)).ToNot(HaveOccurred())

		})

		It("can create/describe/delete admin user - [id:35878]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				var (
					idpType    = "htpasswd"
					idpName    = "myhtpasswd"
					usersValue = "testuser:asCHS-MSV5R-bUwmc-5qb9F"
				)

				By("Create admin")
				output, err := rosaClient.User.CreateAdmin(clusterID)
				if profile.ClusterConfig.ExternalAuthConfig {
					Expect(err).To(HaveOccurred())
					Expect(output.String()).
						To(
							ContainSubstring(`ERR: Creating the 'cluster-admin' user is not supported` +
								` for clusters with external authentication configured`))
					return
				}
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Admin account has been added"))

				By("describe admin")
				output, err = rosaClient.User.DescribeAdmin(clusterID)
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("There is 'cluster-admin' user on cluster"))

				By("List IDP")
				idpTab, _, err := idpService.ListIDP(clusterID)
				Expect(err).To(BeNil())
				Expect(idpTab.IsExist("cluster-admin")).To(BeTrue())

				By("Delete admin")
				output, err = rosaClient.User.DeleteAdmin(clusterID)
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Admin user 'cluster-admin' has been deleted"))

				By("describe admin")
				output, err = rosaClient.User.DescribeAdmin(clusterID)
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("WARN: There is no 'cluster-admin' user on cluster"))

				By("List IDP after the admin is deleted")
				idpTab, _, err = idpService.ListIDP(clusterID)
				Expect(err).To(BeNil())
				Expect(idpTab.IsExist("cluster-admin")).To(BeFalse())

				By("Create one htpasswd idp")
				output, err = idpService.CreateIDP(clusterID, idpName,
					"--type", idpType,
					"--users", usersValue,
					"-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Identity Provider '%s' has been created", idpName))

				By("Create admin")
				output, err = rosaClient.User.CreateAdmin(clusterID)
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Admin account has been added"))
				// Commenting the login part as its consuming long time than estimated for login into cluster
				// blocking the CI jobs. See also below
				// commandOutput := rosaClient.Parser.TextData.Input(output).Parse().Output()
				// command := strings.TrimLeft(commandOutput, " ")
				// command = strings.TrimLeft(command, " ")
				// command = regexp.MustCompile(`[\t\r\n]+`).ReplaceAllString(strings.TrimSpace(command), "\n")

				By("describe admin")
				output, err = rosaClient.User.DescribeAdmin(clusterID)
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("There is 'cluster-admin' user on cluster"))

				By("List IDP")
				idpTab, _, err = idpService.ListIDP(clusterID)
				Expect(err).To(BeNil())
				Expect(idpTab.IsExist("cluster-admin")).To(BeTrue())
				Expect(idpTab.IsExist(idpName)).To(BeTrue())

				// Commenting the login part as its consuming long time than estimated for login into cluster
				// blocking the CI jobs

				// isPrivate, err := rosaClient.Cluster.IsPrivateCluster(clusterID)
				// Expect(err).To(BeNil())

				// if !isPrivate {
				// 	By("login the cluster with the created cluster admin")
				// 	time.Sleep(3 * time.Minute)
				// 	stdout, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
				// 	Expect(err).To(BeNil())
				// 	Expect(stdout.String()).Should(ContainSubstring("Login successful"))
				// }
			})

		It("can create/List/Delete IDPs for rosa clusters - [id:35896]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				// common IDP variables
				var (
					mappingMethod = "claim"
					clientID      = "cccc"
					clientSecret  = "ssss"
				)

				type theIDP struct {
					name string
					url  string // hostedDomain
					org  string
					// ldap
					bindDN            string
					bindPassword      string
					idAttribute       string
					usernameAttribute string
					nameAttribute     string
					emailAttribute    string
					// OpenID
					emailClaims   string
					nameClaims    string
					usernameClaim string
					extraScopes   string
				}

				idp := make(map[string]theIDP)
				idp["Github"] = theIDP{
					name: "mygithub",
					url:  "testhub.com",
					org:  "myorg",
				}
				idp["LDAP"] = theIDP{
					name:              "myldap",
					url:               "ldap://myldap.com",
					bindDN:            "bddn",
					bindPassword:      "bdp",
					idAttribute:       "id",
					usernameAttribute: "usrna",
					nameAttribute:     "na",
					emailAttribute:    "ea",
				}
				idp["Google"] = theIDP{
					name: "mygoogle",
					url:  "google.com",
				}
				idp["Gitlab"] = theIDP{
					name: "mygitlab",
					url:  "https://gitlab.com",
				}
				idp["OpenID"] = theIDP{
					name:          "myopenid",
					url:           "https://google.com",
					emailClaims:   "ec",
					nameClaims:    "nc",
					usernameClaim: "usrnc",
					extraScopes:   "exts",
				}

				By("Create Github IDP")
				output, err := idpService.CreateIDP(clusterID, idp["Github"].name,
					"--mapping-method", mappingMethod,
					"--client-id", clientID,
					"--client-secret", clientSecret,
					"--hostname", idp["Github"].url,
					"--organizations", idp["Github"].org,
					"--type", "github")
				if profile.ClusterConfig.ExternalAuthConfig {
					Expect(err).To(HaveOccurred())
					Expect(output.String()).Should(
						ContainSubstring(`ERR: Adding IDP is not supported for clusters with external authentication configured.`))
					return
				}
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Identity Provider '%s' has been created", idp["Github"].name))

				By("Create Gitlab IDP")
				output, err = idpService.CreateIDP(clusterID, idp["Gitlab"].name,
					"--mapping-method", mappingMethod,
					"--client-id", clientID,
					"--client-secret", clientSecret,
					"--host-url", idp["Gitlab"].url,
					"--organizations", idp["Gitlab"].org,
					"--type", "gitlab")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Identity Provider '%s' has been created", idp["Gitlab"].name))

				By("Create Google IDP")
				output, err = idpService.CreateIDP(clusterID, idp["Google"].name,
					"--mapping-method", mappingMethod,
					"--client-id", clientID,
					"--client-secret", clientSecret,
					"--hosted-domain", idp["Google"].url,
					"--type", "google")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Identity Provider '%s' has been created", idp["Google"].name))

				By("Create LDAP IDP")
				output, err = idpService.CreateIDP(clusterID, idp["LDAP"].name,
					"--mapping-method", mappingMethod,
					"--bind-dn", idp["LDAP"].bindDN,
					"--bind-password", idp["LDAP"].bindPassword,
					"--url", idp["LDAP"].url,
					"--id-attributes", idp["LDAP"].idAttribute,
					"--username-attributes", idp["LDAP"].usernameAttribute,
					"--name-attributes", idp["LDAP"].nameAttribute,
					"--email-attributes", idp["LDAP"].emailAttribute,
					"--insecure",
					"--type", "ldap")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Identity Provider '%s' has been created", idp["LDAP"].name))

				By("Create OpenID IDP")
				output, err = idpService.CreateIDP(clusterID, idp["OpenID"].name,
					"--mapping-method", mappingMethod,
					"--client-id", clientID,
					"--client-secret", clientSecret,
					"--issuer-url", idp["OpenID"].url,
					"--username-claims", idp["OpenID"].usernameClaim,
					"--name-claims", idp["OpenID"].nameClaims,
					"--email-claims", idp["OpenID"].emailClaims,
					"--extra-scopes", idp["OpenID"].extraScopes,
					"--type", "openid")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Identity Provider '%s' has been created", idp["OpenID"].name))

				By("list all IDPs")
				idpTab, _, err := idpService.ListIDP(clusterID)
				Expect(err).To(BeNil())
				for k := range idp {
					Expect(idpTab.IsExist(idp[k].name)).To(BeTrue(), "the idp %s is not in output", idp[k].name)
				}
			})

		It("can create/delete the HTPasswd IDPs - [id:49137]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				var (
					idpType            = "htpasswd"
					idpNames           = []string{"htpasswdn1", "htpasswdn2", "htpasswdn3"}
					singleUserName     string
					singleUserPasswd   string
					multipleuserPasswd []string
				)

				By("Create admin")
				output, err := rosaClient.User.CreateAdmin(clusterID)
				if profile.ClusterConfig.ExternalAuthConfig {
					Expect(err).To(HaveOccurred())
					Expect(output.String()).
						To(
							ContainSubstring(
								`ERR: Creating the 'cluster-admin' user is not supported for clusters ` +
									`with external authentication configured.`))
					return
				}
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Admin account has been added"))

				By("Create one htpasswd idp with multiple users")
				_, singleUserName, singleUserPasswd, err = helper.GenerateHtpasswdPair("user1", "pass1")
				Expect(err).To(BeNil())
				output, err = idpService.CreateIDP(
					clusterID, idpNames[0],
					"--type", idpType,
					"--username", singleUserName,
					"--password", singleUserPasswd,
					"-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				validateIDPOutput(textData, clusterID, idpNames[0])

				By("Create one htpasswd idp with single users")
				multipleuserPasswd, err = helper.GenerateMultipleHtpasswdPairs(2)
				Expect(err).To(BeNil())
				output, err = idpService.CreateIDP(
					clusterID, idpNames[1],
					"--type", idpType,
					"--users", strings.Join(multipleuserPasswd, ","),
					"-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				validateIDPOutput(textData, clusterID, idpNames[1])

				By("Create one htpasswd idp with multiple users from the file")
				multipleuserPasswd, err = helper.GenerateMultipleHtpasswdPairs(3)
				Expect(err).To(BeNil())
				location, err := helper.CreateTempFileWithPrefixAndContent("htpasswdfile", strings.Join(multipleuserPasswd, "\n"))
				Expect(err).To(BeNil())
				defer os.RemoveAll(location)
				output, err = idpService.CreateIDP(
					clusterID, idpNames[2],
					"--type", idpType,
					"--from-file", location,
					"-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				validateIDPOutput(textData, clusterID, idpNames[2])

				By("List IDP")
				idpTab, _, err := idpService.ListIDP(clusterID)
				Expect(err).To(BeNil())
				Expect(idpTab.IsExist("cluster-admin")).To(BeTrue())
				for _, v := range idpNames {
					Expect(idpTab.Idp(v).Type).To(Equal("HTPasswd"))
					Expect(idpTab.Idp(v).AuthURL).To(Equal(""))
				}
			})

		It("Validation for Create/Delete the HTPasswd IDPs by the rosacli command - [id:53031]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				if profile.ClusterConfig.ExternalAuthConfig {
					Skip("Skip this case as IDP is not supported for external auth")
				}
				var (
					idpType         = "htpasswd"
					idpNames        = []string{"htpasswdn1", "htpasswdn2", "htpasswd3"}
					validUserName   = "user1"
					validUserPasswd = "Pass1@htpasswd"
					invalidUserName = "user:2"
				)

				By("Create one htpasswd idp with single user")
				_, validUserName, validUserPasswd, err := helper.GenerateHtpasswdPair(validUserName, validUserPasswd)
				Expect(err).To(BeNil())
				output, err := idpService.CreateIDP(
					clusterID, idpNames[0],
					"--type", idpType,
					"--username", validUserName,
					"--password", validUserPasswd,
					"-y")
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				validateIDPOutput(textData, clusterID, idpNames[0])

				By("Try to create another htpasswd idp with single user")
				_, validUserName, validUserPasswd, err = helper.GenerateHtpasswdPair(validUserName, validUserPasswd)
				Expect(err).To(BeNil())
				output, err = idpService.CreateIDP(
					clusterID, idpNames[1],
					"--type", idpType,
					"--username", validUserName,
					"--password", validUserPasswd,
					"-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				validateIDPOutput(textData, clusterID, idpNames[1])

				By("Delete htpasswd idp")
				output, err = idpService.DeleteIDP(
					clusterID,
					idpNames[0],
				)
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Successfully deleted identity provider '%s' from cluster '%s'", idpNames[0], clusterID))

				By("Try to create htpasswd idp with invalid username")
				output, err = idpService.CreateIDP(
					clusterID, idpNames[2],
					"--type", idpType,
					"--username", invalidUserName,
					"--password", validUserPasswd,
					"-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"invalid username '%s': username must not contain /, :, or %", invalidUserName))
			})

		It("Check the help message and the validation for the IDP creation commands by the rosa cli - [id:38788]",
			labels.Medium, labels.Runtime.Day2, labels.Feature.IDP,
			func() {
				if profile.ClusterConfig.ExternalAuthConfig {
					Skip("IDP is not supported for external auth")
				}
				var (
					idpHtpasswd          = "htpasswd"
					idpGitlab            = "gitlab"
					idpGoogle            = "google"
					idpOpenId            = "openid"
					idpGithub            = "github"
					idpLDAP              = "ldap"
					invalidIdpType       = "invalidIdp"
					UserName             = "user1"
					UserPasswd           = "Pass1@htpasswd"
					invalidMappingMethod = "invalidmappingmethod"
					invalidCaFilePath    = "invalidCaPath"
					invalidTeam          = "invalidTeam"
					invalidLdapUrl       = "ldap.com"
					clientID             = "cccc"
					clientSecret         = "ssss"
				)
				type theIDP struct {
					name string
					url  string // hostedDomain
					org  string
					// OpenID
					emailClaims   string
					nameClaims    string
					usernameClaim string
					extraScopes   string
				}
				idp := make(map[string]theIDP)
				idp["htpasswd"] = theIDP{
					name: "myhtpasswd",
					url:  "htpasswd.com",
					org:  "myorg",
				}
				idp["github"] = theIDP{
					name: "mygithub",
					url:  "testhub.com",
					org:  "myorg",
				}
				idp["gitlab"] = theIDP{
					name: "mygitlab",
					url:  "https://gitlab.com",
					org:  "myorg",
				}
				idp["google"] = theIDP{
					name: "mygoogle",
					url:  "google.com",
				}
				idp["openid"] = theIDP{
					name:          "myopenid",
					url:           "https://google.com",
					emailClaims:   "ec",
					nameClaims:    "nc",
					usernameClaim: "usrnc",
					extraScopes:   "exts",
				}
				idp["ldap"] = theIDP{
					name: "myldap",
					url:  "ldap://myldap.com",
				}

				//Htpasswd
				By("Try creating htpasswd idp with invalid idp type")
				_, UserName, UserPasswd, err := helper.GenerateHtpasswdPair(UserName, UserPasswd)
				Expect(err).To(BeNil())
				output, err := idpService.CreateIDP(
					clusterID, idp[idpHtpasswd].name,
					"--type", invalidIdpType,
					"--username", UserName,
					"--password", UserPasswd,
					"-y")
				Expect(err).NotTo(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Expected a valid IDP type. Options are [github gitlab google htpasswd ldap openid]"))

				//Gitlab
				By("Try creating gitlab idp with invalid idp type")
				output, err = idpService.CreateIDP(clusterID, idp[idpGitlab].name,
					"--client-id", clientID,
					"--client-secret", clientSecret,
					"--host-url", idp[idpGitlab].url,
					"--organizations", idp[idpGitlab].org,
					"--type", invalidIdpType)
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Expected a valid IDP type. Options are [github gitlab google htpasswd ldap openid]"))

				By("Try creating gitlab idp with invalid mapping method")
				output, err = idpService.CreateIDP(clusterID, idp[idpGitlab].name,
					"--mapping-method", invalidMappingMethod,
					"--client-id", clientID,
					"--client-secret", clientSecret,
					"--host-url", idp[idpGitlab].url,
					"--organizations", idp[idpGitlab].org,
					"--type", idpGitlab)
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Failed to create IDP for cluster '%s': Expected a valid mapping method. Options are [add claim generate lookup]",
						clusterID))

				By("Try creating gitlab idp with invalid ca file path")
				output, err = idpService.CreateIDP(clusterID, idp[idpGitlab].name,
					"--client-id", clientID,
					"--client-secret", clientSecret,
					"--host-url", idp[idpGitlab].url,
					"--organizations", idp[idpGitlab].org,
					"--ca", invalidCaFilePath,
					"--type", idpGitlab)
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Failed to create IDP for cluster '%s': Expected a valid certificate bundle: open %s: no such file or directory",
						clusterID,
						invalidCaFilePath))

				//Google --
				By("Try creating google idp with invalid idp type")
				output, err = idpService.CreateIDP(clusterID, idp[idpGoogle].name,
					"--client-id", clientID,
					"--client-secret", clientSecret,
					"--hosted-domain", idp[idpGoogle].url,
					"--organizations", idp[idpGoogle].org,
					"--type", invalidIdpType)
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Expected a valid IDP type. Options are [github gitlab google htpasswd ldap openid]"))

				By("Try creating google idp with invalid mapping method")
				output, err = idpService.CreateIDP(clusterID, idp[idpGoogle].name,
					"--mapping-method", invalidMappingMethod,
					"--client-id", clientID,
					"--client-secret", clientSecret,
					"--hosted-domain", idp[idpGoogle].url,
					"--organizations", idp[idpGoogle].org,
					"--type", idpGoogle)
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Failed to create IDP for cluster '%s': Expected a valid mapping method. Options are [add claim generate lookup]",
						clusterID))

				//OpenId --
				By("Try creating openid idp with invalid idp type")
				output, err = idpService.CreateIDP(clusterID, idp[idpOpenId].name,
					"--client-id", clientID,
					"--client-secret", clientSecret,
					"--issuer-url", idp[idpOpenId].url,
					"--organizations", idp[idpOpenId].org,
					"--type", invalidIdpType)
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Expected a valid IDP type. Options are [github gitlab google htpasswd ldap openid]"))

				By("Try creating openid idp with invalid mapping method")
				output, err = idpService.CreateIDP(clusterID, idp[idpOpenId].name,
					"--mapping-method", invalidMappingMethod,
					"--client-id", clientID,
					"--client-secret", clientSecret,
					"--issuer-url", idp[idpOpenId].url,
					"--username-claims", idp[idpOpenId].usernameClaim,
					"--name-claims", idp[idpOpenId].nameClaims,
					"--email-claims", idp[idpOpenId].emailClaims,
					"--extra-scopes", idp[idpOpenId].extraScopes,
					"--type", idpOpenId)
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Failed to create IDP for cluster '%s': Expected a valid mapping method. Options are [add claim generate lookup]",
						clusterID))

				By("Try creating openid idp with invalid ca file path")
				output, err = idpService.CreateIDP(clusterID, idp[idpOpenId].name,
					"--client-id", clientID,
					"--client-secret", clientSecret,
					"--ca", invalidCaFilePath,
					"--issuer-url", idp[idpOpenId].url,
					"--username-claims", idp[idpOpenId].usernameClaim,
					"--name-claims", idp[idpOpenId].nameClaims,
					"--email-claims", idp[idpOpenId].emailClaims,
					"--extra-scopes", idp[idpOpenId].extraScopes,
					"--type", idpOpenId)
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Failed to create IDP for cluster '%s': Expected a valid certificate bundle: open %s: no such file or directory",
						clusterID,
						invalidCaFilePath))

				//Github --
				By("Create Github IDP with invalid team")
				output, err = idpService.CreateIDP(clusterID, idp[idpGithub].name,
					"--client-id", clientID,
					"--client-secret", clientSecret,
					"--hostname", idp[idpGithub].url,
					"--teams", invalidTeam,
					"--type", idpGithub)
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Failed to add IDP to cluster '%s'",
						clusterID))

				//LDAP --
				By("Create LDAP IDP with invalid url format")
				output, err = idpService.CreateIDP(clusterID, idp[idpLDAP].name,
					"--url", invalidLdapUrl,
					"--insecure",
					"--type", idpLDAP)
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Failed to create IDP for cluster '%s': Expected a valid LDAP URL: parse \"ldap.com\": invalid URI for request",
						clusterID))

				By("Create LDAP IDP with ca and insecure at same time")
				output, err = idpService.CreateIDP(clusterID, idp[idpLDAP].name,
					"--url", idp[idpLDAP].url,
					"--insecure",
					"--ca", invalidCaFilePath,
					"--type", idpLDAP)
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Failed to create IDP for cluster '%s': Cannot use certificate bundle with an insecure connection",
						clusterID))

				By("Try creating LDAP idp with invalid idp type")
				output, err = idpService.CreateIDP(clusterID, idp[idpLDAP].name,
					"--client-id", clientID,
					"--client-secret", clientSecret,
					"--url", idp[idpLDAP].url,
					"--type", invalidIdpType)
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Expected a valid IDP type. Options are [github gitlab google htpasswd ldap openid]"))

				By("Try creating LDAP idp with invalid mapping method")
				output, err = idpService.CreateIDP(clusterID, idp[idpLDAP].name,
					"--client-id", clientID,
					"--client-secret", clientSecret,
					"--mapping-method", invalidMappingMethod,
					"--url", idp[idpLDAP].url,
					"--type", idpLDAP)
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring(
						"Failed to create IDP for cluster '%s': Expected a valid mapping method. Options are [add claim generate lookup]",
						clusterID))

				By("Try creating LDAP idp with invalid ca file path")
				output, err = idpService.CreateIDP(clusterID, idp[idpLDAP].name,
					"--client-id", clientID,
					"--client-secret", clientSecret,
					"--ca", invalidCaFilePath,
					"--url", idp[idpLDAP].url,
					"--type", idpLDAP)
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					Should(ContainSubstring("Failed to create IDP for cluster '%s': Expected a valid certificate bundle: "+
						"open %s: no such file or directory",
						clusterID,
						invalidCaFilePath))
			})

		Context("can Create multiple HTPasswd IDPs including creating admin flow - [id:65160]",
			func() {

				var (
					idpNames   = []string{"htpasswdn1", "htpasswdn2"}
					usersValue = "testuser:as65160-MSV5R-bUwmc-2506"
					idpType    = "htpasswd"
				)

				BeforeEach(func() {
					if profile.ClusterConfig.ExternalAuthConfig {
						Skip("IDP is not supported for external auth")
					}
				})

				It("Create admin first",
					labels.Medium, labels.Runtime.Day2,
					func() {

						By("Create admin")
						output, err := rosaClient.User.CreateAdmin(clusterID)
						Expect(err).To(BeNil())
						textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
						Expect(textData).Should(ContainSubstring("Admin account has been added"))

						By("Create one htpasswd idp with --users flag")
						output, err = idpService.CreateIDP(clusterID, idpNames[0],
							"--type", idpType,
							"--users", usersValue,
							"-y")
						Expect(err).To(BeNil())
						textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
						Expect(textData).Should(ContainSubstring("Identity Provider '%s' has been created", idpNames[0]))

						By("Create one htpasswd idp with --from-file")
						multipleuserPasswd, err := helper.GenerateMultipleHtpasswdPairs(3)
						Expect(err).To(BeNil())
						location, err := helper.CreateTempFileWithPrefixAndContent("htpasswdfile", strings.Join(multipleuserPasswd, "\n"))
						Expect(err).To(BeNil())
						defer os.RemoveAll(location)
						output, err = idpService.CreateIDP(
							clusterID, idpNames[1],
							"--type", idpType,
							"--from-file", location,
							"-y")
						Expect(err).To(BeNil())
						textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
						Expect(textData).Should(ContainSubstring("Identity Provider '%s' has been created", idpNames[1]))

						By("List IDPs")
						idpTab, _, err := idpService.ListIDP(clusterID)
						Expect(err).To(BeNil())
						Expect(idpTab.IsExist("cluster-admin")).To(BeTrue())
						Expect(idpTab.IsExist(idpNames[0])).To(BeTrue())
						Expect(idpTab.IsExist(idpNames[1])).To(BeTrue())

						By("Delete admin")
						output, err = rosaClient.User.DeleteAdmin(clusterID)
						Expect(err).To(BeNil())
						textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
						Expect(textData).Should(ContainSubstring("Admin user 'cluster-admin' has been deleted"))

						By("Delete htpasswd idp")
						for _, idpName := range idpNames {
							output, err = idpService.DeleteIDP(
								clusterID,
								idpName,
							)
							Expect(err).To(BeNil())
							textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
							Expect(textData).Should(ContainSubstring("Successfully deleted identity provider"))
						}

					})

				It("Create admin last",
					labels.Medium, labels.Runtime.Day2,
					func() {

						By("Create one htpasswd idp with --from-file")
						multipleuserPasswd, err := helper.GenerateMultipleHtpasswdPairs(3)
						Expect(err).To(BeNil())
						location, err := helper.CreateTempFileWithPrefixAndContent("htpasswdfile", strings.Join(multipleuserPasswd, "\n"))
						Expect(err).To(BeNil())
						defer os.RemoveAll(location)
						output, err := idpService.CreateIDP(
							clusterID, idpNames[0],
							"--type", idpType,
							"--from-file", location,
							"-y")
						Expect(err).To(BeNil())
						textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
						Expect(textData).Should(ContainSubstring("Identity Provider '%s' has been created", idpNames[0]))

						By("Create htpasswd idp with --users flag")
						output, err = idpService.CreateIDP(clusterID, idpNames[1],
							"--type", idpType,
							"--users", usersValue,
							"-y")
						Expect(err).To(BeNil())
						textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
						Expect(textData).Should(ContainSubstring("Identity Provider '%s' has been created", idpNames[1]))

						By("Create an htpasswd idp with duplicate name")
						output, err = idpService.CreateIDP(clusterID, idpNames[1],
							"--type", idpType,
							"--users", usersValue,
							"-y")
						Expect(err).NotTo(BeNil())
						textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
						Expect(textData).Should(ContainSubstring("Failed to add IDP to cluster '%s'", clusterID))

						By("Create admin")
						output, err = rosaClient.User.CreateAdmin(clusterID)
						Expect(err).To(BeNil())
						textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
						Expect(textData).Should(ContainSubstring("Admin account has been added"))

						By("Create an htpasswd idp with duplicate name 'cluster-admin'")
						output, err = idpService.CreateIDP(clusterID, "cluster-admin",
							"--type", idpType,
							"--users", usersValue,
							"-y")
						Expect(err).NotTo(BeNil())
						textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
						Expect(textData).Should(ContainSubstring("The name \"cluster-admin\" is reserved for admin user IDP"))

						By("List IDPs")
						idpTab, _, err := idpService.ListIDP(clusterID)
						Expect(err).To(BeNil())
						Expect(idpTab.IsExist("cluster-admin")).To(BeTrue())
						Expect(idpTab.IsExist(idpNames[0])).To(BeTrue())
						Expect(idpTab.IsExist(idpNames[1])).To(BeTrue())

						By("Delete admin")
						output, err = rosaClient.User.DeleteAdmin(clusterID)
						Expect(err).To(BeNil())
						textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
						Expect(textData).Should(ContainSubstring("Admin user 'cluster-admin' has been deleted"))

						By("Delete htpasswd idp")
						for _, idpName := range idpNames {
							output, err = idpService.DeleteIDP(
								clusterID,
								idpName,
							)
							Expect(err).To(BeNil())
							textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
							Expect(textData).Should(ContainSubstring("Successfully deleted identity provider"))
						}
					})
			})
	})
