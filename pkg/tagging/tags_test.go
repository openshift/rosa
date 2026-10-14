// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package tagging

import (
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
)

// TestTagging runs the Ginkgo suite for day-2 AWS tag preparation.
func TestTagging(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Day-2 AWS tagging")
}

var _ = Describe("day-2 AWS tag edits", func() {
	cluster := func(hosted bool, version string) *cmv1.Cluster {
		result, err := cmv1.NewCluster().
			Hypershift(cmv1.NewHypershift().Enabled(hosted)).
			Version(cmv1.NewVersion().RawID(version)).Build()
		Expect(err).NotTo(HaveOccurred(), "build hosted=%t version=%q cluster fixture", hosted, version)
		return result
	}

	DescribeTable("control-plane gate",
		func(hosted bool, version string, allowed bool) {
			err := ValidateCluster(cluster(hosted, version))
			if allowed {
				Expect(err).NotTo(HaveOccurred(), "allow hosted=%t version=%q", hosted, version)
			} else {
				Expect(err).To(HaveOccurred(), "reject hosted=%t version=%q", hosted, version)
			}
		},
		Entry("HCP 4.20", true, "4.20.0", true),
		Entry("HCP 4.20 raw ID", true, "openshift-4.20.0", true),
		Entry("HCP 4.20 version ID", true, "openshift-v4.20.0", true),
		Entry("first supported prerelease", true, "4.20.0-0.a", true),
		Entry("older HCP", true, "4.19.99", false),
		Entry("older HCP raw ID", true, "openshift-4.19.99", false),
		Entry("classic", false, "4.20.0", false),
		Entry("unknown version", true, "", false),
		Entry("invalid version", true, "garbage", false),
	)

	It("parses repeated and comma-separated tag arguments", func() {
		tags, err := ParseDesired([]string{`env:prod,team:platform`, `"Cost Center":"Finance Team"`})
		Expect(err).NotTo(HaveOccurred(), "parse repeated, comma-separated, and quoted tags")
		Expect(tags).To(Equal(map[string]string{
			"env": "prod", "team": "platform", "Cost Center": "Finance Team",
		}), "preserve every parsed customer tag")
	})

	It("round-trips spaced keys and values through CLI formatting", func() {
		original := map[string]string{"Cost Center": "Finance Team", "team": "Platform Ops"}
		formatted := aws.FormatUserTagsList(original)
		parsed, err := ParseDesired([]string{formatted})
		Expect(err).NotTo(HaveOccurred(), "parse formatted tags containing spaces")
		Expect(parsed).To(Equal(original), "round-trip spaced keys and values")
	})

	It("uses a single empty flag to clear the set", func() {
		tags, err := ParseDesired([]string{""})
		Expect(err).NotTo(HaveOccurred(), "parse single empty flag as clear")
		Expect(tags).To(BeEmpty(), "empty flag must clear desired tags")
		_, err = ParseDesired([]string{"", "env:prod"})
		Expect(err).To(HaveOccurred(), "reject empty flag combined with another tag")
	})

	DescribeTable("validates interactive tag input before accepting it",
		func(input string, allowed bool) {
			tags, err := parsePromptDesired(input)
			if allowed {
				Expect(err).NotTo(HaveOccurred(), "accept interactive tags %q", input)
				Expect(tags).NotTo(BeNil(), "return desired set for %q", input)
			} else {
				Expect(err).To(HaveOccurred(), "reject interactive tags %q before prompt returns", input)
			}
		},
		Entry("quoted spaces", `"Cost Center":"Finance Team"`, true),
		Entry("clear marker", `""`, true),
		Entry("transformed clear marker", "", true),
		Entry("duplicate key", "owner:one,owner:two", false),
		Entry("protected key", "red-hat-managed:true", false),
		Entry("empty individual value", "owner:", false),
	)

	DescribeTable("rejects invalid customer tags",
		func(value string) {
			_, err := ParseDesired([]string{value})
			Expect(err).To(HaveOccurred(), "reject invalid customer tag input %q", value)
		},
		Entry("individual empty value", "key:"),
		Entry("duplicate key", "key:one,key:two"),
		Entry("AWS key", `"aws:owner":team`),
		Entry("ROSA system key", "red-hat-managed:true"),
		Entry("Kubernetes cluster key", `"kubernetes.io/cluster/test":owned`),
		Entry("OCM system key", `"api.openshift.com/id":test`),
		Entry("invalid value", "owner:#"),
		Entry("key exceeds AWS limit", strings.Repeat("k", 129)+":value"),
		Entry("value exceeds AWS limit", "key:"+strings.Repeat("v", 257)),
	)

	It("diffs updates and null deletions without touching unchanged tags", func() {
		changes := Diff(map[string]string{"env": "old", "delete": "yes", "keep": "same"},
			map[string]string{"env": "new", "add": "yes", "keep": "same"})
		Expect(changes).To(HaveLen(3), "emit only add, update, and delete")
		Expect(*changes["env"]).To(Equal("new"), "update changed tag")
		Expect(*changes["add"]).To(Equal("yes"), "add new tag")
		Expect(changes["delete"]).To(BeNil(), "delete omitted tag with null")
		Expect(Diff(map[string]string{"env": "prod"}, map[string]string{"env": "prod"})).To(BeNil(),
			"unchanged desired tags must produce no diff")
	})

	It("separates pool overrides from inherited cluster tags", func() {
		overrides := PoolOverrides(map[string]string{"team": "platform", "env": "prod"},
			map[string]string{"team": "platform", "env": "stage", "pool": "one"})
		Expect(overrides).To(Equal(map[string]string{"env": "stage", "pool": "one"}),
			"exclude inherited cluster tag and retain pool overrides")
	})

	It("reports requested pool tags that match inherited values", func() {
		var matching []string
		changes, err := PrepareEdit(EditOptions{
			Cluster:     cluster(true, "4.20.0"),
			TagsChanged: true,
			Input:       []string{`team:platform,"Cost Center":"Finance Team",env:stage`},
			ReadCurrent: func() (map[string]string, map[string]string, error) {
				return map[string]string{"team": "platform", "Cost Center": "Finance Team", "env": "prod"},
					map[string]string{"team": "platform", "Cost Center": "Finance Team"}, nil
			},
			OnInheritedMatch: func(key, value string) { matching = append(matching, key+":"+value) },
		})
		Expect(err).NotTo(HaveOccurred(), "prepare pool tag edit with matching inherited values")
		Expect(matching).To(Equal([]string{"Cost Center:Finance Team", "team:platform"}),
			"report only matching inherited tags in stable order")
		Expect(changes).To(HaveLen(3), "clear possible equal-value overrides alongside the pool update")
		Expect(changes).To(HaveKeyWithValue("Cost Center", BeNil()),
			"remove a possible stored override matching the cluster value")
		Expect(changes).To(HaveKeyWithValue("team", BeNil()),
			"remove a possible stored override matching the cluster value")
		Expect(*changes["env"]).To(Equal("stage"), "retain the pool-specific tag update")
	})

	It("sends a removal when clearing an effective tag that may be a stored equal-value override", func() {
		changes, err := PrepareEdit(EditOptions{
			Cluster:     cluster(true, "4.20.0"),
			TagsChanged: true,
			Input:       []string{""},
			ReadCurrent: func() (map[string]string, map[string]string, error) {
				return map[string]string{"team": "platform"}, map[string]string{"team": "platform"}, nil
			},
		})
		Expect(err).NotTo(HaveOccurred(), "prepare clear with an ambiguous inherited pool tag")
		Expect(changes).To(HaveKeyWithValue("team", BeNil()),
			"OCM must receive null to remove a stored pool override with the cluster value")
	})

	It("recognizes tag-only edits with command selectors", func() {
		changed := map[string]bool{"tags": true, "cluster": true}
		isChanged := func(name string) bool { return changed[name] }
		Expect(OnlyTagsChanged(len(changed), isChanged, "cluster")).To(BeTrue(),
			"cluster selection does not make a tag edit a combined edit")
		changed["billing-account"] = true
		Expect(OnlyTagsChanged(len(changed), isChanged, "cluster")).To(BeFalse(),
			"an additional resource edit must prevent the tag-only path")
	})
})
