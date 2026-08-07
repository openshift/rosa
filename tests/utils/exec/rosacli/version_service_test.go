package rosacli

import (
	"github.com/Masterminds/semver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FindBaseAndNearestBackwardMinorVersion", func() {
	It("prefers DEFAULT=yes when that version has a y-1 pair", func() {
		vl := &OpenShiftVersionTableList{
			OpenShiftVersions: []*OpenShiftVersionTableOutput{
				{Version: "5.0.0-ec.5", Default: "no"},
				{Version: "4.22.8", Default: "yes"},
				{Version: "4.21.5", Default: "no"},
			},
		}

		base, backward, err := vl.FindBaseAndNearestBackwardMinorVersion(1, true)

		Expect(err).ToNot(HaveOccurred())
		Expect(base.Version).To(Equal("4.22.8"))
		Expect(backward.Version).To(Equal("4.21.5"))
	})

	It("skips DEFAULT=yes without y-1 and picks the newest version that has a pair", func() {
		vl := &OpenShiftVersionTableList{
			OpenShiftVersions: []*OpenShiftVersionTableOutput{
				{Version: "5.0.0-ec.5", Default: "yes"},
				{Version: "4.22.8", Default: "no"},
				{Version: "4.21.5", Default: "no"},
			},
		}

		base, backward, err := vl.FindBaseAndNearestBackwardMinorVersion(1, true)

		Expect(err).ToNot(HaveOccurred())
		Expect(base.Version).To(Equal("4.22.8"))
		Expect(backward.Version).To(Equal("4.21.5"))
	})

	It("picks the newest version with a y-1 pair when no DEFAULT=yes exists", func() {
		vl := &OpenShiftVersionTableList{
			OpenShiftVersions: []*OpenShiftVersionTableOutput{
				{Version: "5.0.0-ec.5", Default: "no"},
				{Version: "4.22.8", Default: "no"},
				{Version: "4.21.5", Default: "no"},
			},
		}

		base, backward, err := vl.FindBaseAndNearestBackwardMinorVersion(1, true)

		Expect(err).ToNot(HaveOccurred())
		Expect(base.Version).To(Equal("4.22.8"))
		Expect(backward.Version).To(Equal("4.21.5"))
	})

	It("uses Sort fallback to pick newest pair from an ascending list", func() {
		vl := &OpenShiftVersionTableList{
			OpenShiftVersions: []*OpenShiftVersionTableOutput{
				{Version: "4.20.3", Default: "no"},
				{Version: "4.21.5", Default: "no"},
				{Version: "4.22.8", Default: "no"},
			},
		}

		base, backward, err := vl.FindBaseAndNearestBackwardMinorVersion(1, true)

		Expect(err).ToNot(HaveOccurred())
		Expect(base.Version).To(Equal("4.22.8"))
		Expect(backward.Version).To(Equal("4.21.5"))
	})

	It("returns nil when no y-1 pair exists", func() {
		vl := &OpenShiftVersionTableList{
			OpenShiftVersions: []*OpenShiftVersionTableOutput{
				{Version: "5.0.0-ec.5", Default: "no"},
				{Version: "5.0.0-ec.1", Default: "no"},
			},
		}

		base, backward, err := vl.FindBaseAndNearestBackwardMinorVersion(1, true)

		Expect(err).ToNot(HaveOccurred())
		Expect(base).To(BeNil())
		Expect(backward).To(BeNil())
	})

	It("returns nil results for a nil list", func() {
		var vl *OpenShiftVersionTableList

		base, backward, err := vl.FindBaseAndNearestBackwardMinorVersion(1, true)

		Expect(err).ToNot(HaveOccurred())
		Expect(base).To(BeNil())
		Expect(backward).To(BeNil())
	})

	It("returns nil results for an empty list", func() {
		vl := &OpenShiftVersionTableList{}

		base, backward, err := vl.FindBaseAndNearestBackwardMinorVersion(1, true)

		Expect(err).ToNot(HaveOccurred())
		Expect(base).To(BeNil())
		Expect(backward).To(BeNil())
	})

	It("propagates ErrInvalidSemVer when DEFAULT version is malformed", func() {
		vl := &OpenShiftVersionTableList{
			OpenShiftVersions: []*OpenShiftVersionTableOutput{
				{Version: "not-a-version", Default: "yes"},
				{Version: "4.21.5", Default: "no"},
			},
		}

		base, backward, err := vl.FindBaseAndNearestBackwardMinorVersion(1, true)

		Expect(err).To(MatchError(semver.ErrInvalidSemVer))
		Expect(base).To(BeNil())
		Expect(backward).To(BeNil())
	})

	It("propagates ErrInvalidSemVer when DEFAULT is valid but a sibling version is malformed", func() {
		vl := &OpenShiftVersionTableList{
			OpenShiftVersions: []*OpenShiftVersionTableOutput{
				{Version: "4.22.8", Default: "yes"},
				{Version: "not-a-version", Default: "no"},
				{Version: "4.21.5", Default: "no"},
			},
		}

		base, backward, err := vl.FindBaseAndNearestBackwardMinorVersion(1, true)

		Expect(err).To(MatchError(semver.ErrInvalidSemVer))
		Expect(base).To(BeNil())
		Expect(backward).To(BeNil())
	})

	It("propagates ErrInvalidSemVer from Sort during fallback", func() {
		vl := &OpenShiftVersionTableList{
			OpenShiftVersions: []*OpenShiftVersionTableOutput{
				{Version: "not-a-version", Default: "no"},
				{Version: "4.22.8", Default: "no"},
				{Version: "4.21.5", Default: "no"},
			},
		}

		base, backward, err := vl.FindBaseAndNearestBackwardMinorVersion(1, true)

		Expect(err).To(MatchError(semver.ErrInvalidSemVer))
		Expect(base).To(BeNil())
		Expect(backward).To(BeNil())
	})
})
