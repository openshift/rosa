package seaplane

import (
	"context"
	"errors"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"

	seaplanemocks "github.com/openshift/rosa/pkg/seaplane/mocks"
)

func newOidcConfigResolveMocks(ctrl *gomock.Controller) (
	*seaplanemocks.MockInterface, *seaplanemocks.MockOidcConfigInterface,
) {
	seaplaneClient := seaplanemocks.NewMockInterface(ctrl)
	v1 := seaplanemocks.NewMockV1alpha1PublicInterface(ctrl)
	oidcConfigs := seaplanemocks.NewMockOidcConfigInterface(ctrl)
	seaplaneClient.EXPECT().SeaplaneV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().OidcConfigs().Return(oidcConfigs).AnyTimes()
	return seaplaneClient, oidcConfigs
}

var _ = Describe("ResolveOidcConfig", func() {
	ctx := context.Background()
	config := v1alpha1.OidcConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "managed-config", UID: types.UID("oidc-config-uid")},
	}

	It("resolves by server-assigned UID", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, oidcConfigs := newOidcConfigResolveMocks(ctrl)
		oidcConfigs.EXPECT().List(gomock.Any(), platform.ListOptions{}).Return(
			&v1alpha1.OidcConfigList{Items: []v1alpha1.OidcConfig{config}}, nil)

		resolved, err := ResolveOidcConfig(ctx, seaplaneClient, "oidc-config-uid")
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved.Name).To(Equal("managed-config"))
		Expect(string(resolved.UID)).To(Equal("oidc-config-uid"))
	})

	It("resolves by metadata.name", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, oidcConfigs := newOidcConfigResolveMocks(ctrl)
		oidcConfigs.EXPECT().List(gomock.Any(), platform.ListOptions{}).Return(
			&v1alpha1.OidcConfigList{Items: []v1alpha1.OidcConfig{config}}, nil)

		resolved, err := ResolveOidcConfig(ctx, seaplaneClient, "managed-config")
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved.Name).To(Equal("managed-config"))
	})

	It("reports list errors", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, oidcConfigs := newOidcConfigResolveMocks(ctrl)
		oidcConfigs.EXPECT().List(gomock.Any(), platform.ListOptions{}).Return(nil, errors.New("list failed"))

		_, err := ResolveOidcConfig(ctx, seaplaneClient, "oidc-config-uid")
		Expect(err).To(MatchError(ContainSubstring("failed to list OIDC configs")))
	})

	It("reports an unknown config key", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, oidcConfigs := newOidcConfigResolveMocks(ctrl)
		oidcConfigs.EXPECT().List(gomock.Any(), platform.ListOptions{}).Return(&v1alpha1.OidcConfigList{}, nil)

		_, err := ResolveOidcConfig(ctx, seaplaneClient, "missing")
		Expect(err).To(MatchError("OIDC config 'missing' not found"))
	})

	It("requires an ID", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _ := newOidcConfigResolveMocks(ctrl)

		_, err := ResolveOidcConfig(ctx, seaplaneClient, "")
		Expect(err).To(MatchError("OIDC config ID is required"))
	})
})
