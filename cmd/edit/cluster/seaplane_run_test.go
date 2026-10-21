package cluster

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	seaplanemocks "github.com/openshift/rosa/pkg/seaplane/mocks"
	seaplanepathbind "github.com/openshift/rosa/pkg/seaplane/pathbind"
	"github.com/openshift/rosa/pkg/test"
)

func newEditClusterMocks(ctrl *gomock.Controller) (*seaplanemocks.MockInterface, *seaplanemocks.MockClusterInterface) {
	seaplaneClient := seaplanemocks.NewMockInterface(ctrl)
	v1 := seaplanemocks.NewMockV1alpha1PublicInterface(ctrl)
	clusters := seaplanemocks.NewMockClusterInterface(ctrl)
	seaplaneClient.EXPECT().SeaplaneV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().Clusters().Return(clusters).AnyTimes()
	return seaplaneClient, clusters
}

func expectEditProxyGet(clusters *seaplanemocks.MockClusterInterface, response string, statusCode int) {
	if statusCode != http.StatusOK {
		clusters.EXPECT().Get(gomock.Any(), "cluster1", platform.GetOptions{}).
			Return(nil, errors.New("get failed"))
		return
	}
	cluster := &v1alpha1.Cluster{}
	Expect(json.Unmarshal([]byte(response), cluster)).To(Succeed())
	cluster.Name = "cluster1"
	cluster.UID = types.UID("cluster-uid")
	clusters.EXPECT().Get(gomock.Any(), "cluster1", platform.GetOptions{}).Return(cluster, nil)
}

// makeExpirationCmd builds a minimal cobra command with the expiration flag wired
// to args.expirationDuration and optionally set to a value.
func makeExpirationCmd(setExpiration bool) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	cmd.Flags().DurationVar(&args.expirationDuration, "expiration", 0, "")
	if setExpiration {
		if err := cmd.Flags().Set("expiration", "1h"); err != nil {
			panic(err)
		}
	}
	return cmd
}

var _ = Describe("runSeaplaneEdit (cluster)", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
		seaplaneClusterUpdateInput = seaplanepathbindZeroInput()
	})

	AfterEach(func() {
		args.expirationDuration = 0
		args.channelGroup = ""
		args.channel = ""
		args.noProxySlice = nil
		args.httpProxy = ""
		args.httpsProxy = ""
		args.additionalTrustBundleFile = ""
		seaplaneClusterUpdateInput = seaplanepathbindZeroInput()
	})

	It("updates cluster expiration on the success path", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newEditClusterMocks(ctrl)

		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
			Spec:       v1alpha1.ClusterSpec{HostedCluster: v1alpha1.HostedClusterSpecPassthrough{}},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Patch(gomock.Any(), "cluster1", types.MergePatchType, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ types.PatchType, data []byte, _ interface{}) (*v1alpha1.Cluster, error) {
				var body map[string]any
				Expect(json.Unmarshal(data, &body)).To(Succeed())
				spec := body["spec"].(map[string]any)
				Expect(spec).To(HaveKey("expirationTimestamp"))
				Expect(spec).NotTo(HaveKey("hostedCluster"))
				Expect(spec).NotTo(HaveKey("oidcConfigId"))
				return cluster, nil
			})

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		runSeaplaneEdit(t.RosaRuntime, makeExpirationCmd(true))
	})

	It("fails when cluster key is not set", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })
		ocm.SetClusterKey("")
		DeferCleanup(func() { ocm.SetClusterKey("cluster1") })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _ := newEditClusterMocks(ctrl)
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() { runSeaplaneEdit(t.RosaRuntime, makeExpirationCmd(true)) }).To(Panic())
	})

	DescribeTable("updates only the requested no-proxy field",
		func(value, expected string) {
			ctrl := gomock.NewController(GinkgoT())
			seaplaneClient, clusters := newEditClusterMocks(ctrl)
			cluster := &v1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
			}
			clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
				&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
			if expected != "" {
				expectEditProxyGet(clusters, `{"spec":{"hostedCluster":{"configuration":{"proxy":{"httpProxy":"http://proxy.example.com:8080"}}}}}`, http.StatusOK)
			}
			clusters.EXPECT().Patch(gomock.Any(), "cluster1", types.MergePatchType, gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, _ types.PatchType, data []byte, _ interface{}) (*v1alpha1.Cluster, error) {
					expectedBody, err := json.Marshal(map[string]any{
						"spec": map[string]any{
							"hostedCluster": map[string]any{
								"configuration": map[string]any{
									"proxy": map[string]any{"noProxy": expected},
								},
							},
						},
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(data).To(MatchJSON(expectedBody))
					return cluster, nil
				})
			t.RosaRuntime.SeaplaneClient = seaplaneClient
			runSeaplaneEdit(t.RosaRuntime, makeNoProxyCmd(value))
		},
		Entry("sets bypass domains without overwriting proxy URLs", "10.0.0.0/16,.example.com", "10.0.0.0/16,.example.com"),
		Entry("clears bypass domains with an empty value", "", ""),
		Entry("clears bypass domains with the removal marker", `""`, ""),
	)

	DescribeTable("rejects invalid no-proxy edits",
		func(value string) {
			cmd := makeNoProxyCmd(value)
			h := &seaplaneClusterUpdate{cmd: cmd}
			Expect(h.PreRequest(context.Background(), t.RosaRuntime, &seaplaneClusterUpdateInput)).To(HaveOccurred())
		},
		Entry("invalid domain", "invalid domain"),
		Entry("duplicate domain", ".example.com,.example.com"),
	)

	DescribeTable("patches only changed proxy URL fields, including explicit clearing",
		func(flags map[string]string, expected string) {
			ctrl := gomock.NewController(GinkgoT())
			seaplaneClient, clusters := newEditClusterMocks(ctrl)
			cluster := &v1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
			}
			clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
				&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
			clusters.EXPECT().Patch(gomock.Any(), "cluster1", types.MergePatchType, gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, _ types.PatchType, data []byte, _ interface{}) (*v1alpha1.Cluster, error) {
					Expect(data).To(MatchJSON(`{"spec":{"hostedCluster":{"configuration":{"proxy":` + expected + `}}}}`))
					return cluster, nil
				})
			t.RosaRuntime.SeaplaneClient = seaplaneClient
			runSeaplaneEdit(t.RosaRuntime, makeProxyEditCmd(flags))
		},
		Entry("sets HTTP without overwriting HTTPS or bypass domains",
			map[string]string{"http-proxy": "http://proxy.example.com:8080"},
			`{"httpProxy":"http://proxy.example.com:8080"}`),
		Entry("sets HTTPS with an HTTP scheme", map[string]string{"https-proxy": "http://proxy.example.com:8080"},
			`{"httpsProxy":"http://proxy.example.com:8080"}`),
		Entry("sets HTTPS with an HTTPS scheme", map[string]string{"https-proxy": "https://proxy.example.com:8080"},
			`{"httpsProxy":"https://proxy.example.com:8080"}`),
		Entry("clears only HTTP", map[string]string{"http-proxy": ""}, `{"httpProxy":""}`),
		Entry("clears HTTPS with the removal marker", map[string]string{"https-proxy": `""`}, `{"httpsProxy":""}`),
		Entry("clears all proxy fields", map[string]string{"http-proxy": "", "https-proxy": "", "no-proxy": ""},
			`{"httpProxy":"","httpsProxy":"","noProxy":""}`),
	)

	DescribeTable("validates proxy URLs before attempting an API patch",
		func(flag, value, message string) {
			h := &seaplaneClusterUpdate{cmd: makeProxyEditCmd(map[string]string{flag: value})}
			Expect(h.PreRequest(context.Background(), t.RosaRuntime, &seaplaneClusterUpdateInput)).To(
				MatchError(ContainSubstring(message)))
		},
		Entry("HTTP missing scheme", "http-proxy", "invalidvalue", "invalid http-proxy value: URL is missing scheme"),
		Entry("HTTP wrong scheme", "http-proxy", "ftp://test-proxy.com", "URL scheme must be 'http://' or 'https://'"),
		Entry("HTTP with path", "http-proxy", "http://test-proxy.com/extra", "proxy URL should not contain a path '/extra'"),
		Entry("HTTPS missing scheme", "https-proxy", "invalidvalue", "invalid https-proxy value: URL is missing scheme"),
		Entry("HTTPS wrong scheme", "https-proxy", "ftp://test-proxy.com", "URL scheme must be 'http://' or 'https://'"),
		Entry("HTTPS with path", "https-proxy", "https://test-proxy.com/extra", "proxy URL should not contain a path '/extra'"),
	)

	It("rejects no-proxy on a cluster with no proxy URLs", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newEditClusterMocks(ctrl)
		expectEditProxyGet(clusters, `{}`, http.StatusOK)
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		h := &seaplaneClusterUpdate{clusterName: "cluster1", cmd: makeNoProxyCmd("example.com")}
		Expect(h.PreRequest(context.Background(), t.RosaRuntime, &seaplaneClusterUpdateInput)).To(
			MatchError("Expected at least one of the following: http-proxy, https-proxy"))
	})

	It("rejects clearing both proxy URLs while setting bypass domains", func() {
		h := &seaplaneClusterUpdate{cmd: makeProxyEditCmd(map[string]string{
			"http-proxy": "", "https-proxy": "", "no-proxy": "example.com",
		})}
		Expect(h.PreRequest(context.Background(), t.RosaRuntime, &seaplaneClusterUpdateInput)).To(
			MatchError("Failed to update cluster: no-proxy requires http-proxy or https-proxy"))
	})

	It("reports API failures when retrieving existing proxy settings", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newEditClusterMocks(ctrl)
		expectEditProxyGet(clusters, `{"kind":"Status","apiVersion":"v1","status":"Failure","message":"get failed","code":400}`, http.StatusBadRequest)
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		h := &seaplaneClusterUpdate{clusterName: "cluster1", cmd: makeNoProxyCmd("example.com")}
		Expect(h.PreRequest(context.Background(), t.RosaRuntime, &seaplaneClusterUpdateInput)).To(
			MatchError(ContainSubstring("failed to retrieve cluster proxy settings: get failed")))
	})

	It("uses the response proxy projection when checking existing URLs", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newEditClusterMocks(ctrl)
		expectEditProxyGet(clusters, `{"proxy":{"https_proxy":"https://proxy.example.com:8080"}}`, http.StatusOK)
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		h := &seaplaneClusterUpdate{clusterName: "cluster1", cmd: makeNoProxyCmd("example.com")}
		Expect(h.PreRequest(context.Background(), t.RosaRuntime, &seaplaneClusterUpdateInput)).To(Succeed())
	})

	It("validates trust-bundle contents and paths without ignoring the flag", func() {
		bundleFile := filepath.Join(GinkgoT().TempDir(), "bundle.pem")
		Expect(os.WriteFile(bundleFile, []byte("invalid CA"), 0600)).To(Succeed())
		h := &seaplaneClusterUpdate{cmd: makeProxyEditCmd(map[string]string{"additional-trust-bundle-file": bundleFile})}
		Expect(h.PreRequest(context.Background(), t.RosaRuntime, &seaplaneClusterUpdateInput)).To(
			MatchError("Failed to parse additional trust bundle"))
		h.cmd = makeProxyEditCmd(map[string]string{"additional-trust-bundle-file": filepath.Join(GinkgoT().TempDir(), "missing")})
		Expect(h.PreRequest(context.Background(), t.RosaRuntime, &seaplaneClusterUpdateInput)).To(
			MatchError(ContainSubstring("no such file or directory")))
	})

	DescribeTable("clears trust bundles using an explicit empty spec field",
		func(value string) {
			h := &seaplaneClusterUpdate{cmd: makeProxyEditCmd(map[string]string{
				"http-proxy": "http://proxy.example.com:8080", "additional-trust-bundle-file": value,
			})}
			Expect(h.PreRequest(context.Background(), t.RosaRuntime, &seaplaneClusterUpdateInput)).To(Succeed())
			patch, err := h.buildSpecPatch(&seaplaneClusterUpdateInput)
			Expect(err).NotTo(HaveOccurred())
			Expect(patch).To(MatchJSON(`{"spec":{"additionalTrustBundle":"","hostedCluster":{"configuration":{"proxy":{"httpProxy":"http://proxy.example.com:8080"}}}}}`))
		},
		Entry("empty flag value", ""),
		Entry("removal marker", `""`),
	)

	It("patches valid PEM contents at spec.additionalTrustBundle without sending a file path", func() {
		server := httptest.NewTLSServer(nil)
		DeferCleanup(server.Close)
		bundleFile := filepath.Join(GinkgoT().TempDir(), "bundle.pem")
		bundle := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
		Expect(os.WriteFile(bundleFile, bundle, 0600)).To(Succeed())
		h := &seaplaneClusterUpdate{cmd: makeProxyEditCmd(map[string]string{"additional-trust-bundle-file": bundleFile})}
		Expect(h.PreRequest(context.Background(), t.RosaRuntime, &seaplaneClusterUpdateInput)).To(Succeed())
		patch, err := h.buildSpecPatch(&seaplaneClusterUpdateInput)
		Expect(err).NotTo(HaveOccurred())
		expected, err := json.Marshal(map[string]any{"spec": map[string]any{"additionalTrustBundle": string(bundle)}})
		Expect(err).NotTo(HaveOccurred())
		Expect(patch).To(MatchJSON(expected))
	})

	It("rejects a patch when the bundle flag was not prepared", func() {
		h := &seaplaneClusterUpdate{cmd: makeProxyEditCmd(map[string]string{"additional-trust-bundle-file": ""})}
		_, err := h.buildSpecPatch(&seaplaneClusterUpdateInput)
		Expect(err).To(MatchError("additional trust bundle flag was set but value is missing"))
	})

	It("fails when no supported flags are changed", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newEditClusterMocks(ctrl)
		// ResolveClusterName is called before PreRequest validates flags.
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() { runSeaplaneEdit(t.RosaRuntime, makeExpirationCmd(false)) }).To(Panic())
	})

	It("rejects --tags because AWS tags are immutable after creation", func() {
		var exitCode int
		orig := exitFn
		exitFn = func(code int) { exitCode = code; panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newEditClusterMocks(ctrl)
		// ResolveClusterName is called before PreRequest validates flags.
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		t.RosaRuntime.SeaplaneClient = seaplaneClient

		cmd := makeExpirationCmd(true)
		cmd.Flags().String("tags", "", "")
		Expect(cmd.Flags().Set("tags", `{"owner":"platform"}`)).To(Succeed())

		Expect(func() { runSeaplaneEdit(t.RosaRuntime, cmd) }).To(Panic())
		Expect(exitCode).To(Equal(1))
	})

	It("fails when cluster cannot be resolved", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newEditClusterMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{}, nil)

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() { runSeaplaneEdit(t.RosaRuntime, makeExpirationCmd(true)) }).To(Panic())
	})

	It("fails when cluster update fails", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newEditClusterMocks(ctrl)

		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Patch(gomock.Any(), "cluster1", types.MergePatchType, gomock.Any(), gomock.Any()).
			Return(nil, fmt.Errorf("update failed"))

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() { runSeaplaneEdit(t.RosaRuntime, makeExpirationCmd(true)) }).To(Panic())
	})

	It("updates cluster channel group on the success path", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newEditClusterMocks(ctrl)

		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
			Spec:       v1alpha1.ClusterSpec{HostedCluster: v1alpha1.HostedClusterSpecPassthrough{}},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Patch(gomock.Any(), "cluster1", types.MergePatchType, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ types.PatchType, data []byte, _ interface{}) (*v1alpha1.Cluster, error) {
				var body map[string]any
				Expect(json.Unmarshal(data, &body)).To(Succeed())
				spec := body["spec"].(map[string]any)
				hc := spec["hostedCluster"].(map[string]any)
				Expect(hc["channel"]).To(Equal("candidate"))
				props := spec["properties"].(map[string]any)
				Expect(props["channel_group"]).To(Equal("candidate"))
				Expect(spec).NotTo(HaveKey("oidcConfigId"))
				return cluster, nil
			})

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		runSeaplaneEdit(t.RosaRuntime, makeChannelGroupCmd("candidate"))
	})

	It("updates the scheduler profile", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newEditClusterMocks(ctrl)

		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Patch(gomock.Any(), "cluster1", types.MergePatchType, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ types.PatchType, data []byte, _ interface{}) (*v1alpha1.Cluster, error) {
				var body map[string]any
				Expect(json.Unmarshal(data, &body)).To(Succeed())
				spec := body["spec"].(map[string]any)
				hc := spec["hostedCluster"].(map[string]any)
				configuration := hc["configuration"].(map[string]any)
				scheduler := configuration["scheduler"].(map[string]any)
				Expect(scheduler["profile"]).To(Equal("HighNodeUtilization"))
				return cluster, nil
			})

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		runSeaplaneEdit(t.RosaRuntime, makeSchedulerProfileCmd("HighNodeUtilization"))
	})

	It("updates proxy and scheduler settings together without dropping either", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newEditClusterMocks(ctrl)
		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Patch(gomock.Any(), "cluster1", types.MergePatchType, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ types.PatchType, data []byte, _ interface{}) (*v1alpha1.Cluster, error) {
				var body map[string]any
				Expect(json.Unmarshal(data, &body)).To(Succeed())
				spec := body["spec"].(map[string]any)
				hc := spec["hostedCluster"].(map[string]any)
				configuration := hc["configuration"].(map[string]any)
				Expect(configuration).To(HaveLen(2))
				Expect(configuration["scheduler"]).To(Equal(map[string]any{"profile": "HighNodeUtilization"}))
				Expect(configuration["proxy"]).To(Equal(map[string]any{"httpProxy": "http://proxy.example.com:8080"}))
				return cluster, nil
			})
		cmd := makeProxyEditCmd(map[string]string{"http-proxy": "http://proxy.example.com:8080"})
		seaplanepathbind.RegisterClusterUpdateFlags(cmd, &seaplaneClusterUpdateInput)
		Expect(cmd.Flags().Set("scheduler-profile", "HighNodeUtilization")).To(Succeed())
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		runSeaplaneEdit(t.RosaRuntime, cmd)
	})

	It("rejects an invalid scheduler profile", func() {
		cmd := makeSchedulerProfileCmd("NOSCORING")
		h := &seaplaneClusterUpdate{cmd: cmd}
		Expect(h.PreRequest(context.Background(), t.RosaRuntime, &seaplaneClusterUpdateInput)).To(
			MatchError(ContainSubstring("unsupported scheduler profile")))
	})

	It("rejects unsupported channel group", func() {
		Expect(validateSeaplaneChannelArgsFor("fakecg")).To(MatchError(ContainSubstring("Unsupported channel group")))
	})

	It("rejects nightly channel group without a version catalog", func() {
		Expect(validateSeaplaneChannelArgsFor("nightly")).To(MatchError(
			ContainSubstring("is not available for the desired channel group")))
	})
})

func makeChannelGroupCmd(group string) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	cmd.Flags().StringVar(&args.channelGroup, "channel-group", "", "")
	if err := cmd.Flags().Set("channel-group", group); err != nil {
		panic(err)
	}
	return cmd
}

func makeSchedulerProfileCmd(profile string) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	seaplanepathbind.RegisterClusterUpdateFlags(cmd, &seaplaneClusterUpdateInput)
	if err := cmd.Flags().Set("scheduler-profile", profile); err != nil {
		panic(err)
	}
	return cmd
}

func makeNoProxyCmd(value string) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	cmd.Flags().StringSliceVar(&args.noProxySlice, "no-proxy", nil, "")
	Expect(cmd.Flags().Set("no-proxy", value)).To(Succeed())
	return cmd
}

func makeProxyEditCmd(flags map[string]string) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	cmd.Flags().StringVar(&args.httpProxy, "http-proxy", "", "")
	cmd.Flags().StringVar(&args.httpsProxy, "https-proxy", "", "")
	cmd.Flags().StringSliceVar(&args.noProxySlice, "no-proxy", nil, "")
	cmd.Flags().StringVar(&args.additionalTrustBundleFile, "additional-trust-bundle-file", "", "")
	for flag, value := range flags {
		Expect(cmd.Flags().Set(flag, value)).To(Succeed())
	}
	return cmd
}

func validateSeaplaneChannelArgsFor(group string) error {
	orig := args.channelGroup
	args.channelGroup = group
	defer func() { args.channelGroup = orig }()
	return validateSeaplaneChannelArgs()
}

func seaplanepathbindZeroInput() seaplanepathbind.ClusterUpdateInput {
	return seaplanepathbind.ClusterUpdateInput{}
}
