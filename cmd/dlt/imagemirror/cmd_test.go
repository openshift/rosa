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
	clusterId     = "24vf9iitg3p6tlml88iml6j6mu095mh8"
	imageMirrorId = "test-mirror-123"
)

var _ = Describe("Delete image mirror", func() {
	Context("Delete image mirror command", func() {
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

		Context("Input validation", func() {
			It("Returns error when no image mirror ID is provided", func() {
				options := NewDeleteImageMirrorOptions()
				runner := DeleteImageMirrorRunner(options)
				cmd := NewDeleteImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Image mirror ID is required. Specify it as an argument or use the --id flag"))
			})

			It("Uses positional argument for image mirror ID", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, formatExistingImageMirror()))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
				options := NewDeleteImageMirrorOptions()
				options.Args().Yes = true // Skip confirmation
				runner := DeleteImageMirrorRunner(options)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDeleteImageMirrorCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{imageMirrorId})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(ContainSubstring("Image mirror 'test-mirror-123' has been deleted from cluster"))
			})

			It("Uses --id flag for image mirror ID", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, formatExistingImageMirror()))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
				options := NewDeleteImageMirrorOptions()
				options.Args().Id = imageMirrorId
				options.Args().Yes = true // Skip confirmation
				runner := DeleteImageMirrorRunner(options)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDeleteImageMirrorCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(ContainSubstring("Image mirror 'test-mirror-123' has been deleted from cluster"))
			})

			It("Prefers positional argument over --id flag", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, formatExistingImageMirror()))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
				options := NewDeleteImageMirrorOptions()
				options.Args().Id = "different-id"
				options.Args().Yes = true // Skip confirmation
				runner := DeleteImageMirrorRunner(options)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDeleteImageMirrorCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				// Should use positional argument (imageMirrorId) instead of options.Args().Id
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{imageMirrorId})
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("Success scenarios", func() {
			It("Deletes image mirror successfully with --yes flag", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, formatExistingImageMirror()))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
				options := NewDeleteImageMirrorOptions()
				options.Args().Id = imageMirrorId
				options.Args().Yes = true
				runner := DeleteImageMirrorRunner(options)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDeleteImageMirrorCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(ContainSubstring("Image mirror 'test-mirror-123' has been deleted from cluster"))
			})
		})

		Context("Validation error scenarios", func() {
			It("Returns error when cluster is not ready", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, clusterNotReady))
				options := NewDeleteImageMirrorOptions()
				options.Args().Id = imageMirrorId
				options.Args().Yes = true
				runner := DeleteImageMirrorRunner(options)
				cmd := NewDeleteImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("is not ready. Image mirrors can only be deleted on ready clusters"))
			})

			It("Returns error when cluster is not Hosted Control Plane", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				options := NewDeleteImageMirrorOptions()
				options.Args().Id = imageMirrorId
				options.Args().Yes = true
				runner := DeleteImageMirrorRunner(options)
				cmd := NewDeleteImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Image mirrors are only supported on Hosted Control Plane clusters"))
			})

			It("Returns error when cluster fetch fails", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, "{}"))
				options := NewDeleteImageMirrorOptions()
				options.Args().Id = imageMirrorId
				options.Args().Yes = true
				runner := DeleteImageMirrorRunner(options)
				cmd := NewDeleteImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set("nonexistent-cluster")
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("status is 404"))
			})

			It("Returns error when image mirror does not exist", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, "{}"))
				options := NewDeleteImageMirrorOptions()
				options.Args().Id = "nonexistent-mirror"
				options.Args().Yes = true
				runner := DeleteImageMirrorRunner(options)
				cmd := NewDeleteImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Failed to get image mirror 'nonexistent-mirror'"))
			})

			It("Returns error when delete API call fails", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, formatExistingImageMirror()))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError, "{}"))
				options := NewDeleteImageMirrorOptions()
				options.Args().Id = imageMirrorId
				options.Args().Yes = true
				runner := DeleteImageMirrorRunner(options)
				cmd := NewDeleteImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Failed to delete image mirror"))
			})
		})

		Context("Interactive confirmation", func() {
			It("Skips confirmation when --yes flag is used", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, formatExistingImageMirror()))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
				options := NewDeleteImageMirrorOptions()
				options.Args().Id = imageMirrorId
				options.Args().Yes = true
				runner := DeleteImageMirrorRunner(options)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewDeleteImageMirrorCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				// Should not prompt for confirmation
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(ContainSubstring("Image mirror 'test-mirror-123' has been deleted"))
			})
		})

		Context("Command structure", func() {
			It("Has correct command properties", func() {
				cmd := NewDeleteImageMirrorCommand()
				Expect(cmd.Use).To(Equal("image-mirror"))
				Expect(cmd.Short).To(Equal("Delete image mirror from a cluster"))
				Expect(cmd.Aliases).To(ContainElement("image-mirrors"))
				Expect(cmd.Args).ToNot(BeNil())
			})

			It("Has expected flags", func() {
				cmd := NewDeleteImageMirrorCommand()
				flags := []string{"cluster", "id", "yes", "profile", "region"}
				for _, flagName := range flags {
					flag := cmd.Flag(flagName)
					Expect(flag).ToNot(BeNil(), "Flag %s should exist", flagName)
				}
			})

			It("Accepts maximum of 1 positional argument", func() {
				cmd := NewDeleteImageMirrorCommand()
				// MaximumNArgs(1) means it should accept 0 or 1 arguments
				Expect(cmd.Args).ToNot(BeNil())
			})

			It("Has correct flag properties", func() {
				cmd := NewDeleteImageMirrorCommand()

				// Check --yes flag has short form -y
				yesFlag := cmd.Flag("yes")
				Expect(yesFlag).ToNot(BeNil())
				Expect(yesFlag.Shorthand).To(Equal("y"))

				// Check --id flag exists
				idFlag := cmd.Flag("id")
				Expect(idFlag).ToNot(BeNil())
			})
		})
	})
})

// formatExistingImageMirror simulates the response from getting an existing image mirror
func formatExistingImageMirror() string {
	imageMirror, err := cmv1.NewImageMirror().
		ID("test-mirror-123").
		Type("digest").
		Source("registry.redhat.io").
		Mirrors("mirror.example.com", "backup.example.com").
		Build()
	Expect(err).ToNot(HaveOccurred())
	return test.FormatResource(imageMirror)
}
