package admin

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Idps", Ordered, func() {
	var testRuntime test.TestingRuntime
	var clusterKey = "mock-cluster-id"
	var cluster, _ = cmv1.NewCluster().ID(clusterKey).Build()
	var adminIdp, _ = cmv1.NewIdentityProvider().ID("mock-admin-idp-id").Name(ClusterAdminIDPname).
		Type(cmv1.IdentityProviderTypeHtpasswd).Build()
	var nonAdminIdp, _ = cmv1.NewIdentityProvider().ID("mock-nonadmin-idp-id").Name("htpasswd-1").
		Type(ocm.HTPasswdIDPType).Build()
	var adminUser, _ = cmv1.NewHTPasswdUser().Username(ClusterAdminUsername).Build()
	var nonAdminUser, _ = cmv1.NewHTPasswdUser().Username("non-admin").Build()

	BeforeEach(func() {
		testRuntime.InitRuntime()
		testRuntime.RosaRuntime.ClusterKey = clusterKey
	})

	When("FindClusterAdminIDP", func() {
		It("find cluster-amin idp", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatIDPList([]*cmv1.IdentityProvider{adminIdp})))
			existingIdp, err := FindClusterAdminIDP(cluster, testRuntime.RosaRuntime)
			Expect(existingIdp).NotTo(BeNil())
			Expect(err).To(BeNil())
			Expect(existingIdp.Name()).To(Equal(ClusterAdminIDPname))
		})
		It("cannot find cluster-amin idp", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatIDPList([]*cmv1.IdentityProvider{nonAdminIdp})))
			existingIdp, err := FindClusterAdminIDP(cluster, testRuntime.RosaRuntime)
			Expect(existingIdp).To(BeNil())
			Expect(err).To(BeNil())
		})
		It("failed to get idps", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError, ""))
			existingIdp, err := FindClusterAdminIDP(cluster, testRuntime.RosaRuntime)
			Expect(existingIdp).To(BeNil())
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring(
				fmt.Sprintf("Failed to get identity providers for cluster '%s'", clusterKey)))
		})
	})

	When("FindIDPWithAdmin", func() {
		It("find idp with admin user", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatIDPList([]*cmv1.IdentityProvider{adminIdp})))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatHtpasswdUserList([]*cmv1.HTPasswdUser{adminUser})))
			existingIdp, userList, err := FindIDPWithAdmin(cluster, testRuntime.RosaRuntime)
			Expect(existingIdp).NotTo(BeNil())
			Expect(userList).NotTo(BeNil())
			Expect(err).To(BeNil())
			Expect(existingIdp.Name()).To(Equal(ClusterAdminIDPname))
			Expect(userList.Len()).To(Equal(1))
			Expect(userList.Get(0).Username()).To(Equal(ClusterAdminUsername))
		})
		It("cannot find idp with admin user", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatIDPList([]*cmv1.IdentityProvider{nonAdminIdp})))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatHtpasswdUserList([]*cmv1.HTPasswdUser{nonAdminUser})))
			existingIdp, userList, err := FindIDPWithAdmin(cluster, testRuntime.RosaRuntime)
			Expect(existingIdp).To(BeNil())
			Expect(userList).To(BeNil())
			Expect(err).To(BeNil())
		})
		It("failed to get idps", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError, ""))
			existingIdp, userList, err := FindIDPWithAdmin(cluster, testRuntime.RosaRuntime)
			Expect(existingIdp).To(BeNil())
			Expect(userList).To(BeNil())
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring(
				fmt.Sprintf("Failed to get identity providers for cluster '%s'", clusterKey)))
		})
	})
})
