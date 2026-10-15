package hyperfleet

import (
	"context"
	"errors"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
)

func newOidcConfigResolveMocks(ctrl *gomock.Controller) (
	*hfmocks.MockInterface, *hfmocks.MockOidcConfigInterface,
) {
	hf := hfmocks.NewMockInterface(ctrl)
	v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
	oidcConfigs := hfmocks.NewMockOidcConfigInterface(ctrl)
	hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().OidcConfigs().Return(oidcConfigs).AnyTimes()
	return hf, oidcConfigs
}

var _ = Describe("ResolveOidcConfig", func() {
	ctx := context.Background()
	config := v1alpha1.OidcConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "managed-config", UID: types.UID("oidc-config-uid")},
	}

	It("resolves by server-assigned UID", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, oidcConfigs := newOidcConfigResolveMocks(ctrl)
		oidcConfigs.EXPECT().List(gomock.Any(), platform.ListOptions{}).Return(
			&v1alpha1.OidcConfigList{Items: []v1alpha1.OidcConfig{config}}, nil)

		resolved, err := ResolveOidcConfig(ctx, hf, "oidc-config-uid")
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved.Name).To(Equal("managed-config"))
		Expect(string(resolved.UID)).To(Equal("oidc-config-uid"))
	})

	It("resolves by metadata.name", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, oidcConfigs := newOidcConfigResolveMocks(ctrl)
		oidcConfigs.EXPECT().List(gomock.Any(), platform.ListOptions{}).Return(
			&v1alpha1.OidcConfigList{Items: []v1alpha1.OidcConfig{config}}, nil)

		resolved, err := ResolveOidcConfig(ctx, hf, "managed-config")
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved.Name).To(Equal("managed-config"))
	})

	It("reports list errors", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, oidcConfigs := newOidcConfigResolveMocks(ctrl)
		oidcConfigs.EXPECT().List(gomock.Any(), platform.ListOptions{}).Return(nil, errors.New("list failed"))

		_, err := ResolveOidcConfig(ctx, hf, "oidc-config-uid")
		Expect(err).To(MatchError(ContainSubstring("failed to list OIDC configs")))
	})

	It("reports an unknown config key", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, oidcConfigs := newOidcConfigResolveMocks(ctrl)
		oidcConfigs.EXPECT().List(gomock.Any(), platform.ListOptions{}).Return(&v1alpha1.OidcConfigList{}, nil)

		_, err := ResolveOidcConfig(ctx, hf, "missing")
		Expect(err).To(MatchError("OIDC config 'missing' not found"))
	})

	It("requires an ID", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, _ := newOidcConfigResolveMocks(ctrl)

		_, err := ResolveOidcConfig(ctx, hf, "")
		Expect(err).To(MatchError("OIDC config ID is required"))
	})
})
