package imagemirrors

import (
	"context"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	. "github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/test"
)

const (
	clusterId               = "24vf9iitg3p6tlml88iml6j6mu095mh8"
	singleImageMirrorOutput = "ID       TYPE    SOURCE              MIRRORS\n" +
		"mirror1  digest  registry.redhat.io  mirror.example.com\n"
	multipleImageMirrorsOutput = "ID       TYPE    SOURCE              MIRRORS\n" +
		"mirror1  digest  registry.redhat.io  mirror.example.com\n" +
		"mirror2  tag     quay.io/openshift   mirror1.com, mirror2.com\n"
	emptyImageMirrorsMessage = "INFO: No image mirrors found for cluster '24vf9iitg3p6tlml88iml6j6mu095mh8'\n"
)

var _ = Describe("List image mirrors", func() {
	Context("List image mirrors command", func() {
		// Full diff for long string to help debugging
		format.TruncatedDiff = false

		mockClusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
		})

		hypershiftClusterReady := test.FormatClusterList([]*cmv1.Cluster{mockClusterReady})

		singleImageMirrorResponse := formatSingleImageMirror()
		multipleImageMirrorsResponse := formatMultipleImageMirrors()
		emptyImageMirrorsResponse := formatEmptyImageMirrors()

		var t *test.TestingRuntime

		BeforeEach(func() {
			t = test.NewTestRuntime()
			SetOutput("")
		})

		Context("Success scenarios", func() {
			It("Lists single image mirror", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, singleImageMirrorResponse))
				runner := ListImageMirrorsRunner(NewListImageMirrorsOptions())
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListImageMirrorsCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(singleImageMirrorOutput))
			})

			It("Lists multiple image mirrors", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, multipleImageMirrorsResponse))
				runner := ListImageMirrorsRunner(NewListImageMirrorsOptions())
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListImageMirrorsCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(multipleImageMirrorsOutput))
			})

			It("Shows info message when no image mirrors found", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, emptyImageMirrorsResponse))
				runner := ListImageMirrorsRunner(NewListImageMirrorsOptions())
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListImageMirrorsCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(Equal(emptyImageMirrorsMessage))
			})

			It("Outputs JSON when output flag is set", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, singleImageMirrorResponse))
				runner := ListImageMirrorsRunner(NewListImageMirrorsOptions())
				err := t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListImageMirrorsCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = cmd.Flag("output").Value.Set("json")
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				// Verify JSON contains all the image mirror data
				Expect(stdout).To(ContainSubstring("\"id\": \"mirror1\""))
				Expect(stdout).To(ContainSubstring("\"type\": \"digest\""))
				Expect(stdout).To(ContainSubstring("\"source\": \"registry.redhat.io\""))
				Expect(stdout).To(ContainSubstring("\"mirrors\": ["))
				Expect(stdout).To(ContainSubstring("\"mirror.example.com\""))
				// Ensure it's valid JSON array format
				Expect(stdout).To(ContainSubstring("["))
				Expect(stdout).To(ContainSubstring("]"))
			})
		})

		Context("Error scenarios", func() {
			It("Returns error when cluster fetch fails", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, "{}"))
				runner := ListImageMirrorsRunner(NewListImageMirrorsOptions())
				cmd := NewListImageMirrorsCommand()
				err := cmd.Flag("cluster").Value.Set("nonexistent-cluster")
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("status is 404"))
			})

			It("Returns error when ListImageMirrors API call fails", func() {
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError, "{}"))
				runner := ListImageMirrorsRunner(NewListImageMirrorsOptions())
				cmd := NewListImageMirrorsCommand()
				err := cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Failed to list image mirrors"))
			})
		})

		Context("Edge cases", func() {
			It("Handles image mirror with no mirrors array", func() {
				imageMirror, err := cmv1.NewImageMirror().
					ID("mirror-no-mirrors").
					Type("digest").
					Source("registry.example.com").
					Mirrors().
					Build()
				Expect(err).ToNot(HaveOccurred())
				response := fmt.Sprintf("{\n  \"items\": [\n    %s\n  ],\n  \"page\": 0,\n  \"size\": 1,\n  \"total\": 1\n}",
					test.FormatResource(imageMirror))

				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, response))
				runner := ListImageMirrorsRunner(NewListImageMirrorsOptions())
				err = t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListImageMirrorsCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				expectedOutput := "ID                 TYPE    SOURCE                MIRRORS\n" +
					"mirror-no-mirrors  digest  registry.example.com  \n"
				Expect(stdout).To(Equal(expectedOutput))
			})

			It("Handles image mirror with empty mirrors array", func() {
				imageMirror, err := cmv1.NewImageMirror().
					ID("mirror-empty-mirrors").
					Type("tag").
					Source("quay.io/test").
					Mirrors().
					Build()
				Expect(err).ToNot(HaveOccurred())
				response := fmt.Sprintf("{\n  \"items\": [\n    %s\n  ],\n  \"page\": 0,\n  \"size\": 1,\n  \"total\": 1\n}",
					test.FormatResource(imageMirror))

				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, response))
				runner := ListImageMirrorsRunner(NewListImageMirrorsOptions())
				err = t.StdOutReader.Record()
				Expect(err).ToNot(HaveOccurred())
				cmd := NewListImageMirrorsCommand()
				err = cmd.Flag("cluster").Value.Set(clusterId)
				Expect(err).ToNot(HaveOccurred())
				err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
				Expect(err).ToNot(HaveOccurred())
				stdout, err := t.StdOutReader.Read()
				Expect(err).ToNot(HaveOccurred())
				expectedOutput := "ID                    TYPE  SOURCE        MIRRORS\n" +
					"mirror-empty-mirrors  tag   quay.io/test  \n"
				Expect(stdout).To(Equal(expectedOutput))
			})
		})
	})
})

// formatSingleImageMirror simulates the output of APIs for a fake image mirror list with one item
func formatSingleImageMirror() string {
	imageMirror, err := cmv1.NewImageMirror().
		ID("mirror1").
		Type("digest").
		Source("registry.redhat.io").
		Mirrors("mirror.example.com").
		Build()
	Expect(err).ToNot(HaveOccurred())
	return fmt.Sprintf("{\n  \"items\": [\n    %s\n  ],\n  \"page\": 0,\n  \"size\": 1,\n  \"total\": 1\n}",
		test.FormatResource(imageMirror))
}

// formatMultipleImageMirrors simulates the output of APIs for a fake image mirror list with multiple items
func formatMultipleImageMirrors() string {
	imageMirror1, err := cmv1.NewImageMirror().
		ID("mirror1").
		Type("digest").
		Source("registry.redhat.io").
		Mirrors("mirror.example.com").
		Build()
	Expect(err).ToNot(HaveOccurred())

	imageMirror2, err := cmv1.NewImageMirror().
		ID("mirror2").
		Type("tag").
		Source("quay.io/openshift").
		Mirrors("mirror1.com", "mirror2.com").
		Build()
	Expect(err).ToNot(HaveOccurred())

	return fmt.Sprintf("{\n  \"items\": [\n    %s,\n    %s\n  ],\n  \"page\": 0,\n  \"size\": 2,\n  \"total\": 2\n}",
		test.FormatResource(imageMirror1), test.FormatResource(imageMirror2))
}

// formatEmptyImageMirrors simulates the output of APIs for an empty image mirror list
func formatEmptyImageMirrors() string {
	return "{\n  \"items\": [],\n  \"page\": 0,\n  \"size\": 0,\n  \"total\": 0\n}"
}
