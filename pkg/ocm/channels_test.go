package ocm

import (
	"fmt"
	"math"

	"github.com/Masterminds/semver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pkg/Version/Channels", func() {
	DescribeTable("BuildChannelInfo", func(channel, channelgroup string, expected ChannelInfo, wantErr bool, errorString string) {
		info, err := BuildChannelInfo(channel, channelgroup)
		if wantErr {
			Expect(err).To(MatchError(ContainSubstring(errorString)))
			Expect(info).To(BeNil())
			return
		}
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Channel()).To(Equal(expected.Channel()))
		Expect(info.ChannelGroup()).To(Equal(expected.ChannelGroup()))
		Expect(info.SpecifiedChannelGroup()).To(Equal(expected.SpecifiedChannelGroup()))
		Expect(info.YStream()).To(Equal(expected.YStream()))
	},
		Entry("Should fail for misformatted versions",
			fmt.Sprintf("stable-4.%d0", math.MaxInt64), // channel
			"stable",                         // channel-group
			nil,                              // expected ClusterInfo
			true,                             // do we expect an error
			"version parse failure for '4."), // error string
		Entry("Should fail for mismatch between channel group in channel and the one specified in constructor",
			"stable-4.18", // channel
			"eus",         // channel-group
			nil,           // expected ClusterInfo
			true,          // do we expect an error
			"channel_group 'eus' does not match channel group segment of the specified channel 'stable-4.18'"), // error string
		Entry("Should succeed for matched channel and channel_group",
			"candidate-4.18", // channel
			"candidate",      // channel-group
			&channelInfo{
				channel:               "candidate-4.18",
				channelGroup:          "candidate",
				specifiedChannelGroup: "candidate",
				ystream:               *semver.MustParse("4.18"),
			}, // expected ClusterInfo
			false, // do we expect an error
			""),   // error string
		Entry("Should populate unspecified channel_group with group from channel if specified",
			"fast-4.18", // channel
			"",          // channel-group
			&channelInfo{
				channel:               "fast-4.18",
				channelGroup:          "fast",
				specifiedChannelGroup: "",
				ystream:               *semver.MustParse("4.18"),
			}, // expected ClusterInfo
			false, // do we expect an error
			""),   // error string
		Entry("Should succeed for unspecified channel and channel_group. Unspecified channel_group defaults to 'stable'",
			"", // channel
			"", // channel-group
			&channelInfo{
				channelGroup: DefaultChannelGroup,
			}, // expected ClusterInfo
			false, // do we expect an error
			""),   // error string
		Entry("Should succeed for unspecified channel and specified channel_group",
			"",    // channel
			"eus", // channel-group
			&channelInfo{
				channelGroup:          "eus",
				specifiedChannelGroup: "eus",
			}, // expected ClusterInfo
			false, // do we expect an error
			""),   // error string
	)
	DescribeTable("ClusterInfo.ValidForVersion", func(channelInfo ChannelInfo, version string, expectErr bool, errorString string) {
		err := channelInfo.ValidForVersion(version)
		if expectErr {
			Expect(err).To(MatchError(ContainSubstring(errorString)))
			return
		}
		Expect(err).NotTo(HaveOccurred())
	},
		Entry("Should fail when version is higher than channel y-stream",
			&channelInfo{
				channel:      "stable-4.18",
				channelGroup: "stable",
				ystream:      *semver.MustParse("4.18"),
			},
			"4.19.0",
			true,
			"version '4.19.0' is invalid for channel 'stable-4.18': version must be less than or equal to channel version",
		),
		Entry("Should fail when version is invalidly formatted",
			&channelInfo{
				channel:      "stable-4.18",
				channelGroup: "stable",
				ystream:      *semver.MustParse("4.18"),
			},
			"invalid",
			true,
			"version parse failure for 'invalid'",
		),
		Entry("Should succeed for version with same minor as in channel",
			&channelInfo{
				channel:      "stable-4.18",
				channelGroup: "stable",
				ystream:      *semver.MustParse("4.18"),
			},
			"4.18.15",
			false,
			"",
		),
		Entry("Should succeed for version with lower minor than in channel",
			&channelInfo{
				channel:      "stable-4.18",
				channelGroup: "stable",
				ystream:      *semver.MustParse("4.18"),
			},
			"4.17.2",
			false,
			"",
		),
	)
})
