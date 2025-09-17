package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Image Mirror", labels.Feature.ImageMirror, func() {
	defer GinkgoRecover()

	var (
		rosaClient         *rosacli.Client
		imageMirrorService rosacli.ImageMirrorService
		clusterService     rosacli.ClusterService
		clusterID          string
	)

	BeforeEach(func() {
		By("Get the cluster")
		clusterID = config.GetClusterID()
		Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

		By("Init the clients")
		rosaClient = rosacli.NewClient()
		imageMirrorService = rosaClient.ImageMirror
		clusterService = rosaClient.Cluster

		By("Skip testing if the cluster is not a Hosted-cp cluster")
		usHostedCPCluster, err := clusterService.IsHostedCPCluster(clusterID)
		Expect(err).ToNot(HaveOccurred())
		if !usHostedCPCluster {
			SkipNotSTS()
		}
	})

	Context("Image Mirror Management", func() {
		It("can create, list, edit, and delete image mirror on cluster - [id:84761]",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Create a single mirror for the cluster")
				var (
					tMirrorSource = "test.registry/hcp"
					tMirror       = "my.registry.com/testm"
					tType         = "digest"
				)
				output, err := imageMirrorService.CreateImageMirror(
					"-c", clusterID,
					"--source", tMirrorSource,
					"--mirrors", tMirror,
					"--type", tType,
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("has been created on cluster"))
				Expect(output.String()).To(ContainSubstring("Source:"))
				Expect(output.String()).To(ContainSubstring("Mirrors:"))

				By("Create a multiple mirrors for the cluster")
				var (
					tMirrorSource2 = "test.registry/rosa"
					tMirror1       = "my.registry.com/testm"
					tMirror2       = "my.registry.com/nginx"
				)
				output, err = imageMirrorService.CreateImageMirror(
					"-c", clusterID,
					"--source", tMirrorSource2,
					"--mirrors", fmt.Sprintf("%s,%s", tMirror1, tMirror2),
				)
				Expect(err).ToNot(HaveOccurred())

				By("List the image mirrors")
				output, err = imageMirrorService.ListImageMirror(
					"-c", clusterID,
				)
				Expect(err).ToNot(HaveOccurred())
				imageMirrorsList, err := imageMirrorService.ReflectImageMirrorList(output)
				Expect(err).ToNot(HaveOccurred())

				imageMirror := imageMirrorsList.GetImageMirrorBySource(tMirrorSource)

				Expect(imageMirror.ID).ToNot(BeEmpty())
				imageMirrorID := imageMirror.ID
				Expect(imageMirror.Type).To(Equal(tType))
				Expect(imageMirror.Source).To(Equal(tMirrorSource))
				Expect(imageMirror.Mirrors).To(ContainSubstring(tMirror))

				imageMirror2 := imageMirrorsList.GetImageMirrorBySource(tMirrorSource2)
				Expect(imageMirror2.ID).ToNot(BeEmpty())
				imageMirrorID2 := imageMirror2.ID
				Expect(imageMirror2.Type).To(Equal("digest"))
				Expect(imageMirror2.Source).To(Equal(tMirrorSource2))
				Expect(imageMirror2.Mirrors).To(ContainSubstring(tMirror1))
				Expect(imageMirror2.Mirrors).To(ContainSubstring(tMirror2))

				By("Validation for creating image mirror")
				output, err = imageMirrorService.CreateImageMirror(
					"--source", "source/validation",
					"--mirrors", tMirror,
					"--type", tType,
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("required flag(s) \"cluster\" not set"))

				output, err = imageMirrorService.CreateImageMirror(
					"-c", clusterID,
					"--source", "source/validation",
					"--type", tType,
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("required flag(s) \"mirrors\" not set"))

				output, err = imageMirrorService.CreateImageMirror(
					"-c", clusterID,
					"--mirrors", tMirror,
					"--type", tType,
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("required flag(s) \"source\" not set"))

				output, err = imageMirrorService.CreateImageMirror(
					"-c", "notexistcid",
					"--source", "source/validation",
					"--mirrors", tMirror,
					"--type", tType,
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("There is no cluster with identifier or name"))

				output, err = imageMirrorService.CreateImageMirror(
					"-c", clusterID,
					"--source", tMirrorSource,
					"--mirrors", tMirror,
					"--type", tType,
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("already exists for cluster"))

				output, err = imageMirrorService.CreateImageMirror(
					"-c", clusterID,
					"--source", "source/validation",
					"--mirrors", tMirror,
					"--type", "deafult",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("type must be 'digest' if specified"))

				By("Edit the single mirror to multiple mirrors")
				updatedMirrorVal1 := "11/22"
				updatedMirrorVal2 := "rosa/cli"
				output, err = imageMirrorService.EditImageMirror(
					"-c", clusterID,
					"--id", imageMirrorID,
					"--mirrors", fmt.Sprintf("%s,%s", updatedMirrorVal1, updatedMirrorVal2),
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("has been updated on cluster"))
				Expect(output.String()).To(ContainSubstring("Source:"))
				Expect(output.String()).To(ContainSubstring("Updated mirrors"))

				By("Edit the multiple mirror to single mirror")
				updatedMirrorVal := "rosa/cli/singemirror"
				output, err = imageMirrorService.EditImageMirror(
					"-c", clusterID,
					"--id", imageMirrorID2,
					"--mirrors", updatedMirrorVal,
				)
				Expect(err).ToNot(HaveOccurred())

				By("Check the update work")
				output, err = imageMirrorService.ListImageMirror(
					"-c", clusterID,
				)
				Expect(err).ToNot(HaveOccurred())
				imageMirrorsList, err = imageMirrorService.ReflectImageMirrorList(output)
				Expect(err).ToNot(HaveOccurred())

				imageMirror = imageMirrorsList.GetImageMirrorById(imageMirrorID)
				Expect(imageMirror.Mirrors).To(ContainSubstring(updatedMirrorVal1))
				Expect(imageMirror.Mirrors).To(ContainSubstring(updatedMirrorVal2))

				imageMirror = imageMirrorsList.GetImageMirrorById(imageMirrorID2)
				Expect(imageMirror.Mirrors).To(ContainSubstring(updatedMirrorVal))

				By("Validation for editing image mirror")
				output, err = imageMirrorService.EditImageMirror(
					"--id", imageMirrorID,
					"--mirrors", tMirror,
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("required flag(s) \"cluster\" not set"))

				output, err = imageMirrorService.EditImageMirror(
					"-c", clusterID,
					"--id", imageMirrorID,
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("required flag(s) \"mirrors\" not set"))

				output, err = imageMirrorService.EditImageMirror(
					"-c", clusterID,
					"--mirrors", tMirror,
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("Image mirror ID is required"))

				output, err = imageMirrorService.EditImageMirror(
					"-c", clusterID,
					"--id", "notexistedid",
					"--mirrors", tMirror,
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("not found"))

				output, err = imageMirrorService.EditImageMirror(
					"-c", "notexistcid",
					"--id", imageMirrorID,
					"--mirrors", tMirror,
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("There is no cluster with identifier or name"))

				By("Validation for deleting image mirror")
				output, err = imageMirrorService.DeleteImageMirror(
					"--id", imageMirrorID,
					"-y",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("required flag(s) \"cluster\" not set"))

				output, err = imageMirrorService.DeleteImageMirror(
					"-c", clusterID,
					"-y",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("Image mirror ID is required"))

				output, err = imageMirrorService.DeleteImageMirror(
					"-c", clusterID,
					"--id", "notexistedid",
					"-y",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("not found"))

				output, err = imageMirrorService.DeleteImageMirror(
					"-c", "notexistcid",
					"--id", imageMirrorID,
					"-y",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("There is no cluster with identifier or name"))

				By("Delete the mirrors")
				output, err = imageMirrorService.DeleteImageMirror(
					"-c", clusterID,
					"--id", imageMirrorID,
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("has been deleted from cluster"))

				output, err = imageMirrorService.DeleteImageMirror(
					"-c", clusterID,
					"--id", imageMirrorID2,
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("has been deleted from cluster"))

				By("Check the mirrors deleted")
				output, err = imageMirrorService.ListImageMirror(
					"-c", clusterID,
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("No image mirrors found for cluster"))
			})
	})
})
