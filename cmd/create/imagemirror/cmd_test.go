package imagemirror

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/test"
)

const (
	clusterId = "24vf9iitg3p6tlml88iml6j6mu095mh8"
)

var _ = Describe("Create image mirror", func() {
	Context("Create image mirror command", func() {
		mockHCPClusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
		})

		mockClassicCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(false))
		})

		mockClusterNotReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateInstalling)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
		})

		hcpClusterReady := test.FormatClusterList([]*cmv1.Cluster{mockHCPClusterReady})
		classicClusterReady := test.FormatClusterList([]*cmv1.Cluster{mockClassicCluster})
		clusterNotReady := test.FormatClusterList([]*cmv1.Cluster{mockClusterNotReady})

		var t *test.TestingRuntime

		BeforeEach(func() {
			t = test.NewTestRuntime()
		})

		Context("Success scenarios", func() {
			It("Creates image mirror successfully with all required parameters", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusCreated, formatCreatedImageMirror()))
				options := NewCreateImageMirrorOptions()
				options.Args().Source = "registry.redhat.io"
				options.Args().Mirrors = []string{"mirror.example.com", "backup.example.com"}
				runner := CreateImageMirrorRunner(options)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewCreateImageMirrorCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(ContainSubstring("Image mirror with ID 'test-mirror-123' has been created on cluster"))
				Expect(stdout).To(ContainSubstring("Source: registry.redhat.io"))
				Expect(stdout).To(ContainSubstring("Mirrors: [mirror.example.com backup.example.com]"))
			})

			It("Creates image mirror with custom type", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusCreated, formatCreatedImageMirror()))
				options := NewCreateImageMirrorOptions()
				options.Args().Type = "digest"
				options.Args().Source = "quay.io/openshift"
				options.Args().Mirrors = []string{"internal.corp.com/openshift"}
				runner := CreateImageMirrorRunner(options)
				cmd := NewCreateImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("Creates image mirror with single mirror", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusCreated, formatCreatedImageMirror()))
				options := NewCreateImageMirrorOptions()
				options.Args().Source = "docker.io/library"
				options.Args().Mirrors = []string{"mirror.company.com/dockerhub"}
				runner := CreateImageMirrorRunner(options)
				cmd := NewCreateImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("Validation error scenarios", func() {
			It("Returns error when cluster is not ready", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, clusterNotReady))
				options := NewCreateImageMirrorOptions()
				options.Args().Source = "registry.redhat.io"
				options.Args().Mirrors = []string{"mirror.example.com"}
				runner := CreateImageMirrorRunner(options)
				cmd := NewCreateImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("is not ready. Image mirrors can only be created on ready clusters"))
			})

			It("Returns error when cluster is not Hosted Control Plane", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				options := NewCreateImageMirrorOptions()
				options.Args().Source = "registry.redhat.io"
				options.Args().Mirrors = []string{"mirror.example.com"}
				runner := CreateImageMirrorRunner(options)
				cmd := NewCreateImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Image mirrors are only supported on Hosted Control Plane clusters"))
			})

			It("Returns error when cluster fetch fails", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, "{}"))
				options := NewCreateImageMirrorOptions()
				options.Args().Source = "registry.redhat.io"
				options.Args().Mirrors = []string{"mirror.example.com"}
				runner := CreateImageMirrorRunner(options)
				cmd := NewCreateImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set("nonexistent-cluster")
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("status is 404"))
			})

			It("Returns error when CreateImageMirror API call fails", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError, "{}"))
				options := NewCreateImageMirrorOptions()
				options.Args().Source = "registry.redhat.io"
				options.Args().Mirrors = []string{"mirror.example.com"}
				runner := CreateImageMirrorRunner(options)
				cmd := NewCreateImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Failed to create image mirror"))
			})
		})

		Context("Runtime validation", func() {
			It("Returns error when mirrors array is empty", func() {
				// Test the runtime validation that checks if mirrors slice is empty
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				options := NewCreateImageMirrorOptions()
				options.Args().Source = "registry.redhat.io"
				options.Args().Mirrors = []string{} // Empty array
				runner := CreateImageMirrorRunner(options)
				cmd := NewCreateImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("At least one mirror registry must be specified"))
			})

			It("Has default type as digest", func() {
				cmd := NewCreateImageMirrorCommand()
				typeFlag := cmd.Flag("type")
				Expect(typeFlag.DefValue).To(Equal("digest"))
			})
		})

		Context("Command structure", func() {
			It("Has correct command properties", func() {
				cmd := NewCreateImageMirrorCommand()
				Expect(cmd.Use).To(Equal("image-mirror"))
				Expect(cmd.Short).To(Equal("Create image mirror for a cluster"))
				Expect(cmd.Aliases).To(ContainElement("image-mirrors"))
				Expect(cmd.Args).ToNot(BeNil())
			})

			It("Has expected flags", func() {
				cmd := NewCreateImageMirrorCommand()
				flags := []string{"cluster", "type", "source", "mirrors", "profile", "region"}
				for _, flagName := range flags {
					flag := cmd.Flag(flagName)
					Expect(flag).ToNot(BeNil(), "Flag %s should exist", flagName)
				}
			})
		})
	})
})

// formatCreatedImageMirror simulates the response from creating an image mirror
func formatCreatedImageMirror() string {
	imageMirror, err := cmv1.NewImageMirror().
		ID("test-mirror-123").
		Type("digest").
		Source("registry.redhat.io").
		Mirrors("mirror.example.com", "backup.example.com").
		Build()
	Expect(err).ToNot(HaveOccurred())
	return test.FormatResource(imageMirror)
}
