package cluster

import (
	"encoding/json"
	"io"
	"os"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("runHyperfleetList structured output", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
		output.SetOutput("")
		DeferCleanup(output.SetOutput, "")
	})

	stubClusterList := func(ctrl *gomock.Controller, clusters []v1alpha1.Cluster) {
		hf := hfmocks.NewMockInterface(ctrl)
		v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
		clusterIface := hfmocks.NewMockClusterInterface(ctrl)
		hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
		v1.EXPECT().Clusters().Return(clusterIface).AnyTimes()
		clusterIface.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: clusters}, nil)
		t.RosaRuntime.HyperFleetClient = hf
	}

	captureStdout := func(f func()) string {
		r, w, _ := os.Pipe()
		orig := os.Stdout
		os.Stdout = w
		f()
		w.Close()
		os.Stdout = orig
		out, _ := io.ReadAll(r)
		return string(out)
	}

	testClusters := []v1alpha1.Cluster{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "my-cluster", UID: types.UID("cluster-uid-1")},
			Status:     v1alpha1.ClusterStatus{Phase: v1alpha1.ClusterPhaseReady},
		},
	}

	It("outputs valid JSON with expected fields when --output json is set", func() {
		ctrl := gomock.NewController(GinkgoT())
		stubClusterList(ctrl, testClusters)
		output.SetOutput(output.JSON)

		stdout := captureStdout(func() { runHyperfleetList(t.RosaRuntime) })

		var items []clusterListItem
		Expect(json.Unmarshal([]byte(stdout), &items)).To(Succeed())
		Expect(items).To(HaveLen(1))
		Expect(items[0].ID).To(Equal("cluster-uid-1"))
		Expect(items[0].Name).To(Equal("my-cluster"))
		Expect(items[0].Topology).To(Equal("Hosted CP"))
	})

	It("outputs YAML with expected fields when --output yaml is set", func() {
		ctrl := gomock.NewController(GinkgoT())
		stubClusterList(ctrl, testClusters)
		output.SetOutput(output.YAML)

		stdout := captureStdout(func() { runHyperfleetList(t.RosaRuntime) })

		Expect(stdout).To(ContainSubstring("name: my-cluster"))
		Expect(stdout).To(ContainSubstring("id: cluster-uid-1"))
		Expect(stdout).To(ContainSubstring("topology: Hosted CP"))
	})

	It("outputs [] in JSON mode when no clusters exist", func() {
		ctrl := gomock.NewController(GinkgoT())
		stubClusterList(ctrl, nil)
		output.SetOutput(output.JSON)

		stdout := captureStdout(func() { runHyperfleetList(t.RosaRuntime) })

		Expect(stdout).To(ContainSubstring("[]"))
	})
})
