/*
Copyright (c) 2025 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ocm

import (
	"reflect"
	"testing"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func TestBuildLogForwarder(t *testing.T) {
	tests := []struct {
		name   string
		input  *LogForwarderConfig
		verify func(t *testing.T, out *cmv1.LogForwarder)
	}{
		{
			name:  "nil config returns empty builder",
			input: nil,
			verify: func(t *testing.T, out *cmv1.LogForwarder) {
				if len(out.Applications()) != 0 {
					t.Errorf("expected no applications, got %+v", out.Applications())
				}
				if cw, ok := out.GetCloudwatch(); ok {
					t.Errorf("expected no CloudWatch config, got %+v", cw)
				}
				if groups, ok := out.GetGroups(); ok && len(groups) > 0 {
					t.Errorf("expected no groups, got %+v", groups)
				}
				if s3, ok := out.GetS3(); ok {
					t.Errorf("expected no S3 config, got %+v", s3)
				}
			},
		},
		{
			name: "full config populates builder",
			input: &LogForwarderConfig{
				Applications:           []string{"app1", "app2"},
				CloudWatchLogGroupName: "cw-group",
				CloudWatchLogRoleArn:   "cw-arn",
				GroupsLogVersion:       []string{"v1", "v2"},
				S3ConfigBucketName:     "my-bucket",
				S3ConfigBucketPrefix:   "logs/",
			},
			verify: func(t *testing.T, out *cmv1.LogForwarder) {
				if !reflect.DeepEqual(out.Applications(), []string{"app1", "app2"}) {
					t.Errorf("applications mismatch: %+v", out.Applications())
				}

				if cw, ok := out.GetCloudwatch(); !ok {
					t.Errorf("expected CloudWatch config")
				} else {
					if cw.LogGroupName() != "cw-group" {
						t.Errorf("cw group mismatch: %v", cw.LogGroupName())
					}
					if cw.LogDistributionRoleArn() != "cw-arn" {
						t.Errorf("cw arn mismatch: %v", cw.LogDistributionRoleArn())
					}
				}

				if groups, ok := out.GetGroups(); !ok || len(groups) != 2 {
					t.Fatalf("expected 2 groups but got %+v", groups)
				} else {
					got := []string{groups[0].Version(), groups[1].Version()}
					expected := []string{"v1", "v2"}
					if !reflect.DeepEqual(got, expected) {
						t.Errorf("group versions mismatch: %v", got)
					}
				}

				if s3, ok := out.GetS3(); !ok {
					t.Errorf("expected S3 config")
				} else {
					if s3.BucketName() != "my-bucket" {
						t.Errorf("s3 bucket mismatch: %v", s3.BucketName())
					}
					if s3.BucketPrefix() != "logs/" {
						t.Errorf("s3 prefix mismatch: %v", s3.BucketPrefix())
					}
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			builder := BuildLogForwader(tc.input)
			out, err := builder.Build()
			if err != nil {
				t.Fatalf("failed to build logForwarder: %v", err)
			}
			tc.verify(t, out)
		})
	}
}

func TestGetLogForwardConfig(t *testing.T) {
	t.Run("returns nil when input is nil", func(t *testing.T) {
		cfg := GetLogForwardConfig(nil)
		if cfg != nil {
			t.Errorf("expected nil, got %+v", cfg)
		}
	})

	t.Run("extracts config from LogForwarder object", func(t *testing.T) {
		lf, err := cmv1.NewLogForwarder().
			Applications("a1", "a2").
			Cloudwatch(cmv1.NewLogForwarderCloudWatchConfig().
				LogGroupName("cw-group").
				LogDistributionRoleArn("cw-arn")).
			Groups(
				cmv1.NewLogForwarderGroup().Version("v1"),
				cmv1.NewLogForwarderGroup().Version("v2"),
			).
			S3(cmv1.NewLogForwarderS3Config().BucketName("bucket").BucketPrefix("prefix/")).
			Build()
		if err != nil {
			t.Fatalf("failed to build lf: %v", err)
		}

		cfg := GetLogForwardConfig(lf)

		if !reflect.DeepEqual(cfg.Applications, []string{"a1", "a2"}) {
			t.Errorf("applications mismatch: %+v", cfg.Applications)
		}
		if cfg.CloudWatchLogGroupName != "cw-group" {
			t.Errorf("cw group mismatch: %s", cfg.CloudWatchLogGroupName)
		}
		if cfg.CloudWatchLogRoleArn != "cw-arn" {
			t.Errorf("cw arn mismatch: %s", cfg.CloudWatchLogRoleArn)
		}
		if !reflect.DeepEqual(cfg.GroupsLogVersion, []string{"v1", "v2"}) {
			t.Errorf("group versions mismatch: %+v", cfg.GroupsLogVersion)
		}
		if cfg.S3ConfigBucketName != "bucket" {
			t.Errorf("s3 bucket mismatch: %s", cfg.S3ConfigBucketName)
		}
		if cfg.S3ConfigBucketPrefix != "prefix/" {
			t.Errorf("s3 prefix mismatch: %s", cfg.S3ConfigBucketPrefix)
		}
	})
}
