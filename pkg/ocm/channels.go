package ocm

import (
	"cmp"
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
)

var channelRegex = regexp.MustCompile(`^(stable|eus|nightly|fast|candidate)-\d+\.\d+$`)

// enforces valid configuration of Channel and ChannelGroup when used in tandem
type ChannelInfo interface {
	// returns the channel specified upon ChannelInfo creation if applicable (otherwise the empty string)
	Channel() string
	// returns the channelGroup specified upon ChannelInfo creation, if applicable (otherwise the empty string)
	SpecifiedChannelGroup() string
	// returns the channelGroup as calculated from  the channel, if applicable (otherwise the empty string)
	ChannelGroup() string
	// returns the y-stream version (major-minor) calculated from the channel, if applicable (otherswise the empty string)
	YStream() string
	// returns an error if the OCP version specified is invalid for this Channel+ChannelGroup configuration
	ValidForVersion(version string) error
}

type channelInfo struct {
	channel               string
	channelGroup          string
	specifiedChannelGroup string
	ystream               semver.Version
}

func parseChannel(channel string) (group string, ystream string, err error) {
	if !channelRegex.MatchString(channel) {
		return "", "",
			fmt.Errorf("channel '%s' does not match proper channel format (<group>-<version>, i.e. stable-4.18)", channel)
	}
	segments := strings.Split(channel, "-")
	return segments[0], segments[1], nil
}

// smart constructor that returns a ChannelInfo object only if the configuration is valid.
// otherwise returns an error
func BuildChannelInfo(channel, channelGroup string) (ChannelInfo, error) {
	if channel == "" {
		return &channelInfo{
			specifiedChannelGroup: channelGroup,
			channelGroup:          cmp.Or(channelGroup, DefaultChannelGroup),
		}, nil
	}
	group, ystream, error := parseChannel(channel)
	if error != nil {
		return nil, error
	}
	ystreamVer, err := semver.NewVersion(ystream)
	if err != nil {
		return nil, fmt.Errorf("version parse failure for '%s': %w", ystream, err)
	}
	if channelGroup != "" && group != channelGroup {
		return nil, fmt.Errorf(
			"channel_group '%s' does not match channel group segment of the specified channel '%s'", channelGroup, channel)
	}
	return &channelInfo{
		channelGroup:          group,
		specifiedChannelGroup: channelGroup,
		channel:               channel,
		ystream:               *ystreamVer,
	}, nil
}

// Mock constructor: please only use for testing
func NewMockChannelInfo(channel, channelGroup string) ChannelInfo {
	return &channelInfo{
		specifiedChannelGroup: channelGroup,
		channelGroup:          channelGroup,
		channel:               channel,
	}
}

func (i *channelInfo) ValidForVersion(version string) error {
	ver, err := semver.NewVersion(version)
	if err != nil {
		return fmt.Errorf("version parse failure for '%s': %w", version, err)
	}
	if ver.Minor() > i.ystream.Minor() {
		return fmt.Errorf(
			"version '%s' is invalid for channel '%s': version must be less than or equal to channel version",
			version, i.channel)
	}
	return nil
}

func (i *channelInfo) Channel() string {
	return i.channel
}

func (i *channelInfo) ChannelGroup() string {
	return i.channelGroup
}

func (i *channelInfo) SpecifiedChannelGroup() string {
	return i.specifiedChannelGroup
}

func (i *channelInfo) YStream() string {
	return i.ystream.String()
}
