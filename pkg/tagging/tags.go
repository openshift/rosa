// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

// Package tagging prepares day-2 ROSA HCP AWS tag edits for the OCM API.
package tagging

import (
	"fmt"
	"strings"

	commontags "github.com/openshift-online/ocm-common/pkg/aws/tags"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/helper/versions"
	"github.com/openshift/rosa/pkg/interactive" //nolint:depguard
)

// MinimalVersion is the first control-plane version that reconciles AWS tag edits in place.
const MinimalVersion = "4.20.0-0.a"

// EditOptions supplies the command input and resource-specific tag read for a day-2 edit.
type EditOptions struct {
	// Cluster is the cluster whose control-plane version governs the edit.
	Cluster *cmv1.Cluster
	// TagsChanged reports whether the user explicitly supplied --tags.
	TagsChanged bool
	// Input contains the values supplied through --tags.
	Input []string
	// OptInQuestion asks whether to edit tags in interactive mode.
	OptInQuestion string
	// DesiredQuestion asks for the complete desired tag set.
	DesiredQuestion string
	// Help is the --tags flag help displayed with either prompt.
	Help string
	// OptInErrorContext names the operation in an opt-in prompt error.
	OptInErrorContext string
	// DesiredErrorContext names the operation in a tag-value prompt error.
	DesiredErrorContext string
	// ReadCurrent returns the current customer tags and optional inherited cluster tags.
	// It is called only after the cluster passes the version and product gate.
	ReadCurrent func() (current, inherited map[string]string, err error)
}

// ValidateCluster rejects tag edits outside ROSA HCP 4.20 and later.
func ValidateCluster(cluster *cmv1.Cluster) error {
	if cluster == nil || cluster.Hypershift() == nil || !aws.IsHostedCP(cluster) {
		return fmt.Errorf("editing AWS tags is only supported for Hosted Control Plane clusters")
	}
	if cluster.Version() == nil || cluster.Version().RawID() == "" {
		return fmt.Errorf("cannot edit AWS tags without a control-plane version")
	}
	supported, err := SupportsControlPlaneVersion(cluster.Version().RawID())
	if err != nil {
		return fmt.Errorf("cannot check control-plane version for AWS tag editing: %w", err)
	}
	if !supported {
		return fmt.Errorf("editing AWS tags requires OpenShift version 4.20.0 or greater")
	}
	return nil
}

// SupportsControlPlaneVersion reports whether a raw OCM control-plane version supports day-2 AWS tags.
// It accepts both plain versions and the prefixed raw IDs returned by OCM.
func SupportsControlPlaneVersion(rawID string) (bool, error) {
	version := strings.TrimPrefix(rawID, "openshift-v")
	version = strings.TrimPrefix(version, "openshift-")
	return versions.IsGreaterThanOrEqual(version, MinimalVersion)
}

// ParseDesired parses the complete desired customer tag set. A single empty flag value clears it.
func ParseDesired(input []string) (map[string]string, error) {
	if len(input) == 1 && input[0] == "" {
		return map[string]string{}, nil
	}
	for _, item := range input {
		if item == "" {
			return nil, fmt.Errorf("an empty --tags value must be used alone to remove all tags")
		}
	}
	desired, err := aws.ParseUserTags(input, aws.UserTagOptions{})
	if err != nil {
		return nil, err
	}
	if err := commontags.Validate(desired); err != nil {
		return nil, err
	}
	for key := range desired {
		if strings.HasPrefix(key, "api.openshift.com/") {
			return nil, fmt.Errorf("AWS tag key %q is reserved", key)
		}
	}
	return desired, nil
}

// PromptDesired asks for the complete desired tag set and validates each answer before accepting it.
// Pressing Enter keeps the displayed default; entering two quotes removes all editable tags.
func PromptDesired(question, help string, defaults map[string]string) (map[string]string, error) {
	input, err := interactive.GetString(interactive.Input{
		Question: question,
		Help:     help,
		Default:  aws.FormatUserTagsList(defaults),
		Validators: []interactive.Validator{func(value interface{}) error {
			text, ok := value.(string)
			if !ok {
				return fmt.Errorf("AWS tags must be a string")
			}
			_, err := parsePromptDesired(text)
			return err
		}},
	})
	if err != nil {
		return nil, err
	}
	return parsePromptDesired(input)
}

// PrepareEdit validates eligibility, collects the desired tags, and diffs them against current tags.
// ReadCurrent supplies inherited cluster tags for node pools and nil inheritance for clusters.
func PrepareEdit(options EditOptions) (map[string]*string, error) {
	requested := options.TagsChanged
	if !requested && interactive.Enabled() && ValidateCluster(options.Cluster) == nil {
		var err error
		requested, err = interactive.GetBool(interactive.Input{
			Question: options.OptInQuestion,
			Default:  false,
			Help:     options.Help,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to prompt for %s: %w", options.OptInErrorContext, err)
		}
	}
	if !requested {
		return nil, nil
	}
	if err := ValidateCluster(options.Cluster); err != nil {
		return nil, err
	}
	if options.ReadCurrent == nil {
		return nil, fmt.Errorf("cannot edit AWS tags without a current-tag reader")
	}

	var desired map[string]string
	if options.TagsChanged {
		var err error
		desired, err = ParseDesired(options.Input)
		if err != nil {
			return nil, fmt.Errorf("invalid AWS tags: %w", err)
		}
	}
	current, inherited, err := options.ReadCurrent()
	if err != nil {
		return nil, err
	}
	if inherited != nil {
		current = PoolOverrides(inherited, current)
	}
	if interactive.Enabled() {
		promptDefault := current
		if options.TagsChanged {
			promptDefault = desired
		}
		desired, err = PromptDesired(options.DesiredQuestion, options.Help, promptDefault)
		if err != nil {
			return nil, fmt.Errorf("failed to prompt for %s: %w", options.DesiredErrorContext, err)
		}
	}
	if inherited != nil {
		desired = PoolOverrides(inherited, desired)
	}
	return Diff(current, desired), nil
}

// OnlyTagsChanged reports whether all changed flags are tags or allowed command options.
// changedFlag is the command's flag-change lookup; ignored names are resource selectors
// and options that do not edit the resource.
func OnlyTagsChanged(changedCount int, changedFlag func(string) bool, ignored ...string) bool {
	count := 0
	for _, name := range append([]string{"tags"}, ignored...) {
		if changedFlag(name) {
			count++
		}
	}
	return count == changedCount
}

// parsePromptDesired handles the literal clear marker before and after the prompt transformer.
func parsePromptDesired(input string) (map[string]string, error) {
	if input == "" || input == `""` {
		return ParseDesired([]string{""})
	}
	return ParseDesired(aws.SplitUserTagList(input))
}

// Diff returns the PATCH tag changes needed to replace current with desired.
// Nil return means that the sets are equal; nil values in the result delete keys.
func Diff(current, desired map[string]string) map[string]*string {
	if commontags.Equal(current, desired) {
		return nil
	}
	changes := commontags.ComputeDiff(current, desired)
	patch := make(map[string]*string, len(changes.Add)+len(changes.Update)+len(changes.Remove))
	for key, value := range changes.Add {
		value := value
		patch[key] = &value
	}
	for key, value := range changes.Update {
		value := value
		patch[key] = &value
	}
	for _, key := range changes.Remove {
		patch[key] = nil
	}
	return patch
}

// PoolOverrides removes inherited cluster tags from the current node-pool tag view.
// An explicit override equal to the inherited value is indistinguishable and redundant.
func PoolOverrides(clusterTags, poolTags map[string]string) map[string]string {
	overrides := make(map[string]string)
	for key, value := range poolTags {
		if inherited, ok := clusterTags[key]; !ok || inherited != value {
			overrides[key] = value
		}
	}
	return overrides
}
