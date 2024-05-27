package e2e

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Edit IDP",
	labels.Day2,
	labels.FeatureIDP,
	func() {
		defer GinkgoRecover()

		var (
			clusterID  string
			rosaClient *rosacli.Client
			idpService rosacli.IDPService
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the clients")
			rosaClient = rosacli.NewClient()
			idpService = rosaClient.IDP

		})

		AfterEach(func() {
			By("Clean remaining resources")
			var errorList []error
			errorList = append(errorList, rosaClient.CleanResources(clusterID))
			Expect(errors.Join(errorList...)).ToNot(HaveOccurred())

		})

		It("can create/describe/delete admin user - [id:35878]",
			labels.Critical,
			func() {
				var (
					idpType    = "htpasswd"
					idpName    = "myhtpasswd"
					usersValue = "testuser:asCHS-MSV5R-bUwmc-5qb9F"
				)

				By("Create admin")
				output, err := rosaClient.User.CreateAdmin(clusterID)
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
				commandOutput := rosaClient.Parser.TextData.Input(output).Parse().Output()
				command := strings.TrimLeft(commandOutput, " ")
				command = strings.TrimLeft(command, " ")
				command = regexp.MustCompile(`[\t\r\n]+`).ReplaceAllString(strings.TrimSpace(command), "\n")

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

				By("login the cluster with the created cluster admin")
				time.Sleep(3 * time.Minute)
				stdout, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
				Expect(err).To(BeNil())
				Expect(stdout.String()).Should(ContainSubstring("Login successful"))
			})

		It("can create/List/Delete IDPs for rosa clusters - [id:35896]",
			labels.Critical,
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
			labels.Critical,
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
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Admin account has been added"))

				By("Create one htpasswd idp with multiple users")
				_, singleUserName, singleUserPasswd, err = common.GenerateHtpasswdPair("user1", "pass1")
				Expect(err).To(BeNil())
				output, err = idpService.CreateIDP(
					clusterID, idpNames[0],
					"--type", idpType,
					"--username", singleUserName,
					"--password", singleUserPasswd,
					"-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Identity Provider '%s' has been created", idpNames[0]))
				Expect(textData).Should(ContainSubstring("To log in to the console, open"))
				Expect(textData).Should(ContainSubstring("and click on '%s'", idpNames[0]))

				By("Create one htpasswd idp with single users")
				multipleuserPasswd, err = common.GenerateMultipleHtpasswdPairs(2)
				Expect(err).To(BeNil())
				output, err = idpService.CreateIDP(
					clusterID, idpNames[1],
					"--type", idpType,
					"--users", strings.Join(multipleuserPasswd, ","),
					"-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Identity Provider '%s' has been created", idpNames[1]))
				Expect(textData).Should(ContainSubstring("To log in to the console, open"))
				Expect(textData).Should(ContainSubstring("and click on '%s'", idpNames[1]))

				By("Create one htpasswd idp with multiple users from the file")
				multipleuserPasswd, err = common.GenerateMultipleHtpasswdPairs(3)
				Expect(err).To(BeNil())
				location, err := common.CreateTempFileWithPrefixAndContent("htpasswdfile", strings.Join(multipleuserPasswd, "\n"))
				Expect(err).To(BeNil())
				defer os.RemoveAll(location)
				output, err = idpService.CreateIDP(
					clusterID, idpNames[2],
					"--type", idpType,
					"--from-file", location,
					"-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Identity Provider '%s' has been created", idpNames[2]))
				Expect(textData).Should(ContainSubstring("To log in to the console, open"))
				Expect(textData).Should(ContainSubstring("and click on '%s'", idpNames[2]))

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
			labels.Critical,
			func() {
				var (
					idpType         = "htpasswd"
					idpNames        = []string{"htpasswdn1", "htpasswdn2", "htpasswd3"}
					validUserName   = "user1"
					validUserPasswd = "Pass1@htpasswd"
					invalidUserName = "user:2"
				)

				By("Create one htpasswd idp with single user")
				_, validUserName, validUserPasswd, err := common.GenerateHtpasswdPair(validUserName, validUserPasswd)
				Expect(err).To(BeNil())
				output, err := idpService.CreateIDP(
					clusterID, idpNames[0],
					"--type", idpType,
					"--username", validUserName,
					"--password", validUserPasswd,
					"-y")
				Expect(err).To(BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Identity Provider '%s' has been created", idpNames[0]))
				Expect(textData).Should(ContainSubstring("To log in to the console, open"))
				Expect(textData).Should(ContainSubstring("and click on '%s'", idpNames[0]))

				By("Try to create another htpasswd idp with single user")
				_, validUserName, validUserPasswd, err = common.GenerateHtpasswdPair(validUserName, validUserPasswd)
				Expect(err).To(BeNil())
				output, err = idpService.CreateIDP(
					clusterID, idpNames[1],
					"--type", idpType,
					"--username", validUserName,
					"--password", validUserPasswd,
					"-y")
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Identity Provider '%s' has been created", idpNames[1]))
				Expect(textData).Should(ContainSubstring("To log in to the console, open"))
				Expect(textData).Should(ContainSubstring("and click on '%s'", idpNames[1]))

				By("Delete htpasswd idp")
				output, err = idpService.DeleteIDP(
					clusterID,
					idpNames[0],
				)
				Expect(err).To(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Successfully deleted identity provider '%s' from cluster '%s'", idpNames[0], clusterID))

				By("Try to create htpasswd idp with invalid username")
				output, err = idpService.CreateIDP(
					clusterID, idpNames[2],
					"--type", idpType,
					"--username", invalidUserName,
					"--password", validUserPasswd,
					"-y")
				Expect(err).NotTo(BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).Should(ContainSubstring(fmt.Sprintf("Failed to add IDP to cluster '%s': Invalid username '%s': Username must not contain /, :, or %%", clusterID, invalidUserName)))
			})
	})
