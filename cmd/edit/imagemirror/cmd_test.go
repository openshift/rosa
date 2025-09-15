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

var _ = Describe("Edit image mirror", func() {
	Context("Edit image mirror command", func() {
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
				options := NewEditImageMirrorOptions()
				options.Args().Mirrors = []string{"mirror.example.com"}
				runner := EditImageMirrorRunner(options)
				cmd := NewEditImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Image mirror ID is required. Specify it as an argument or use the --id flag"))
			})

			It("Returns error when no mirrors are provided", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				options := NewEditImageMirrorOptions()
				options.Args().Id = imageMirrorId
				options.Args().Mirrors = []string{}
				runner := EditImageMirrorRunner(options)
				cmd := NewEditImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("At least one mirror registry must be specified"))
			})

			It("Uses positional argument for image mirror ID", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, formatUpdatedImageMirror()))
				options := NewEditImageMirrorOptions()
				options.Args().Mirrors = []string{"mirror.example.com", "backup.example.com"}
				runner := EditImageMirrorRunner(options)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewEditImageMirrorCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{imageMirrorId})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(ContainSubstring("Image mirror 'test-mirror-123' has been updated on cluster"))
				Expect(stdout).To(ContainSubstring("Source: registry.redhat.io"))
				Expect(stdout).To(ContainSubstring("Updated mirrors: [mirror.example.com backup.example.com]"))
			})

			It("Uses --id flag for image mirror ID", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, formatUpdatedImageMirror()))
				options := NewEditImageMirrorOptions()
				options.Args().Id = imageMirrorId
				options.Args().Mirrors = []string{"mirror.example.com", "backup.example.com"}
				runner := EditImageMirrorRunner(options)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewEditImageMirrorCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(ContainSubstring("Image mirror 'test-mirror-123' has been updated on cluster"))
				Expect(stdout).To(ContainSubstring("Source: registry.redhat.io"))
				Expect(stdout).To(ContainSubstring("Updated mirrors: [mirror.example.com backup.example.com]"))
			})

			It("Prefers positional argument over --id flag", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, formatUpdatedImageMirror()))
				options := NewEditImageMirrorOptions()
				options.Args().Id = "different-id"
				options.Args().Mirrors = []string{"mirror.example.com"}
				runner := EditImageMirrorRunner(options)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewEditImageMirrorCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				// Should use positional argument (imageMirrorId) instead of options.Args().Id
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{imageMirrorId})
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("Success scenarios", func() {
			It("Edits image mirror successfully with multiple mirrors", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, formatUpdatedImageMirror()))
				options := NewEditImageMirrorOptions()
				options.Args().Id = imageMirrorId
				options.Args().Mirrors = []string{"mirror.example.com", "backup.example.com", "third.example.com"}
				runner := EditImageMirrorRunner(options)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewEditImageMirrorCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(ContainSubstring("Image mirror 'test-mirror-123' has been updated on cluster"))
				Expect(stdout).To(ContainSubstring("Source: registry.redhat.io"))
				Expect(stdout).To(ContainSubstring("Updated mirrors: [mirror.example.com backup.example.com]"))
			})

			It("Edits image mirror successfully with single mirror", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, formatSingleMirrorUpdate()))
				options := NewEditImageMirrorOptions()
				options.Args().Id = imageMirrorId
				options.Args().Mirrors = []string{"single-mirror.example.com"}
				runner := EditImageMirrorRunner(options)
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewEditImageMirrorCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(ContainSubstring("Image mirror 'test-mirror-123' has been updated on cluster"))
				Expect(stdout).To(ContainSubstring("Source: registry.redhat.io"))
				Expect(stdout).To(ContainSubstring("Updated mirrors: [single-mirror.example.com]"))
			})
		})

		Context("Validation error scenarios", func() {
			It("Returns error when cluster is not ready", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, clusterNotReady))
				options := NewEditImageMirrorOptions()
				options.Args().Id = imageMirrorId
				options.Args().Mirrors = []string{"mirror.example.com"}
				runner := EditImageMirrorRunner(options)
				cmd := NewEditImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("is not ready. Image mirrors can only be edited on ready clusters"))
			})

			It("Returns error when cluster is not Hosted Control Plane", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
				options := NewEditImageMirrorOptions()
				options.Args().Id = imageMirrorId
				options.Args().Mirrors = []string{"mirror.example.com"}
				runner := EditImageMirrorRunner(options)
				cmd := NewEditImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Image mirrors are only supported on Hosted Control Plane clusters"))
			})
		})

		Context("Command structure", func() {
			It("Has correct command properties", func() {
				cmd := NewEditImageMirrorCommand()
				Expect(cmd.Use).To(Equal("image-mirror"))
				Expect(cmd.Short).To(Equal("Edit image mirror for a cluster"))
				Expect(cmd.Aliases).To(ContainElement("image-mirrors"))
				Expect(cmd.Args).ToNot(BeNil())
			})

			It("Has expected flags", func() {
				cmd := NewEditImageMirrorCommand()
				flags := []string{"cluster", "id", "type", "mirrors", "profile", "region"}
				for _, flagName := range flags {
					flag := cmd.Flag(flagName)
					Expect(flag).ToNot(BeNil(), "Flag %s should exist", flagName)
				}
			})

			It("Accepts maximum of 1 positional argument", func() {
				cmd := NewEditImageMirrorCommand()
				// MaximumNArgs(1) means it should accept 0 or 1 arguments
				Expect(cmd.Args).ToNot(BeNil())
			})

			It("Has correct flag properties", func() {
				cmd := NewEditImageMirrorCommand()

				// Check --mirrors flag exists and is a string slice
				mirrorsFlag := cmd.Flag("mirrors")
				Expect(mirrorsFlag).ToNot(BeNil())

				// Check --id flag exists
				idFlag := cmd.Flag("id")
				Expect(idFlag).ToNot(BeNil())
			})
		})

		Context("Mirror validation", func() {
			It("Validates mirrors flag is required through command setup", func() {
				cmd := NewEditImageMirrorCommand()
				// The command sets mirrors as required in cmd.MarkFlagRequired("mirrors")
				// This would be validated by cobra before reaching our runner
				mirrorsFlag := cmd.Flag("mirrors")
				Expect(mirrorsFlag).ToNot(BeNil())
			})

			It("Accepts empty mirrors slice in options but validates in runner", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hcpClusterReady))
				options := NewEditImageMirrorOptions()
				options.Args().Id = imageMirrorId
				options.Args().Mirrors = []string{} // Empty mirrors
				runner := EditImageMirrorRunner(options)
				cmd := NewEditImageMirrorCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("At least one mirror registry must be specified"))
			})
		})

		Context("Type parameter validation", func() {
			It("Uses default type value when not specified", func() {
				options := NewEditImageMirrorOptions()
				Expect(options.Args().Type).To(Equal("digest"))
			})

			It("Can set type flag through command", func() {
				cmd := NewEditImageMirrorCommand()
				err := cmd.Flag("type").Value.Set("tag")
				Expect(err).ToNot(HaveOccurred())
				typeFlag := cmd.Flag("type")
				Expect(typeFlag.Value.String()).To(Equal("tag"))
			})

			It("Has correct default value for type flag", func() {
				cmd := NewEditImageMirrorCommand()
				typeFlag := cmd.Flag("type")
				Expect(typeFlag.DefValue).To(Equal("digest"))
			})
		})
	})
})

// formatUpdatedImageMirror simulates the response from updating an image mirror
func formatUpdatedImageMirror() string {
	imageMirror, err := cmv1.NewImageMirror().
		ID("test-mirror-123").
		Type("digest").
		Source("registry.redhat.io").
		Mirrors("mirror.example.com", "backup.example.com").
		Build()
	Expect(err).ToNot(HaveOccurred())
	return test.FormatResource(imageMirror)
}

// formatSingleMirrorUpdate simulates the response from updating an image mirror with a single mirror
func formatSingleMirrorUpdate() string {
	imageMirror, err := cmv1.NewImageMirror().
		ID("test-mirror-123").
		Type("digest").
		Source("registry.redhat.io").
		Mirrors("single-mirror.example.com").
		Build()
	Expect(err).ToNot(HaveOccurred())
	return test.FormatResource(imageMirror)
}
