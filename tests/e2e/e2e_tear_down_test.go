package e2e

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/handler"
)

var _ = Describe("Cluster destroy", labels.Feature.Cluster, func() {
	It("by profile",
		labels.Runtime.Destroy,
		labels.Critical,
		func() {
			client := rosacli.NewClient()
			profile := handler.LoadProfileYamlFileByENV()
			clusterHandler, err := handler.NewClusterHandlerFromFilesystem(client, profile)
			Expect(err).ToNot(HaveOccurred())
			var errs = clusterHandler.Destroy()
			Expect(len(errs)).To(Equal(0), fmt.Sprintf("Errors while destroying the cluster: %v", errors.Join(errs...)))
		})
})

var _ = Describe("Cluster destroy on Konflux", labels.Feature.Cluster, func() {
	It("by profile",
		labels.Runtime.DestroyOnKonflux,
		labels.Critical,
		func() {
			client := rosacli.NewClient()
			profile := handler.LoadProfileYamlFileByENV()
			clusterHandler, err := handler.NewClusterHandlerForKonflux(client, profile)
			Expect(err).ToNot(HaveOccurred())
			var errs = clusterHandler.Destroy()
			Expect(len(errs)).To(Equal(0), fmt.Sprintf("Errors while destroying the cluster: %v", errors.Join(errs...)))
		})
})
