package pathbind

import (
	"context"
	"testing"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/pathbind"
	"github.com/spf13/cobra"
)

type clusterCreateBoolField struct {
	flag string
	set  func(*ClusterCreateInput, *bool)
	get  func(*ClusterCreateInput) *bool
}

var clusterCreateBoolFields = []clusterCreateBoolField{
	{
		flag: "delete-protection",
		set:  func(i *ClusterCreateInput, p *bool) { i.DeleteProtection = p },
		get:  func(i *ClusterCreateInput) *bool { return i.DeleteProtection },
	},
	{
		flag: "serialize-image-pulls",
		set:  func(i *ClusterCreateInput, p *bool) { i.SerializeImagePulls = p },
		get:  func(i *ClusterCreateInput) *bool { return i.SerializeImagePulls },
	},
	{
		flag: "multi-arch",
		set:  func(i *ClusterCreateInput, p *bool) { i.MultiArch = p },
		get:  func(i *ClusterCreateInput) *bool { return i.MultiArch },
	},
}

type clusterUpdateBoolField struct {
	flag string
	set  func(*ClusterUpdateInput, *bool)
	get  func(*ClusterUpdateInput) *bool
}

var clusterUpdateBoolFields = []clusterUpdateBoolField{
	{
		flag: "delete-protection",
		set:  func(i *ClusterUpdateInput, p *bool) { i.DeleteProtection = p },
		get:  func(i *ClusterUpdateInput) *bool { return i.DeleteProtection },
	},
	{
		flag: "serialize-image-pulls",
		set:  func(i *ClusterUpdateInput, p *bool) { i.SerializeImagePulls = p },
		get:  func(i *ClusterUpdateInput) *bool { return i.SerializeImagePulls },
	},
	{
		flag: "multi-arch",
		set:  func(i *ClusterUpdateInput, p *bool) { i.MultiArch = p },
		get:  func(i *ClusterUpdateInput) *bool { return i.MultiArch },
	},
}

func TestNormalizeClusterCreateInput_clearsUnsetBoolFlags(t *testing.T) {
	for _, field := range clusterCreateBoolFields {
		t.Run(field.flag, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Bool(field.flag, false, "")

			input := &ClusterCreateInput{}
			field.set(input, new(bool))
			normalizeClusterCreateInput(cmd, input)

			if field.get(input) != nil {
				t.Fatalf("expected nil %s, got %v", field.flag, *field.get(input))
			}
		})
	}
}

func TestNormalizeClusterCreateInput_keepsChangedBoolFlags(t *testing.T) {
	for _, field := range clusterCreateBoolFields {
		t.Run(field.flag, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Bool(field.flag, false, "")
			if err := cmd.Flags().Set(field.flag, "true"); err != nil {
				t.Fatalf("Set flag: %v", err)
			}

			input := &ClusterCreateInput{}
			v := true
			field.set(input, &v)
			normalizeClusterCreateInput(cmd, input)

			got := field.get(input)
			if got == nil || !*got {
				t.Fatalf("expected %s=true after explicit flag set", field.flag)
			}
		})
	}
}

func TestNormalizeClusterCreateInput_clearsUnsetSerializeImagePullsFromExpand(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("serialize-image-pulls", false, "")

	input := &ClusterCreateInput{
		SerializeImagePulls: new(bool),
	}
	normalizeClusterCreateInput(cmd, input)

	obj := &v1alpha1.Cluster{}
	if err := pathbind.Expand(context.Background(), *input, obj); err != nil {
		t.Fatalf("Expand failed: %v", err)
	}
	cfg := obj.Spec.HostedCluster.Configuration
	if cfg != nil && cfg.Kubelet != nil && cfg.Kubelet.SerializeImagePulls != nil {
		t.Fatal("expected serializeImagePulls omitted from expanded cluster spec")
	}
}

func TestNormalizeClusterUpdateInput_clearsUnsetBoolFlags(t *testing.T) {
	for _, field := range clusterUpdateBoolFields {
		t.Run(field.flag, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Bool(field.flag, false, "")

			input := &ClusterUpdateInput{}
			field.set(input, new(bool))
			normalizeClusterUpdateInput(cmd, input)

			if field.get(input) != nil {
				t.Fatalf("expected nil %s, got %v", field.flag, *field.get(input))
			}
		})
	}
}

func TestNormalizeClusterUpdateInput_keepsChangedBoolFlags(t *testing.T) {
	for _, field := range clusterUpdateBoolFields {
		t.Run(field.flag, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Bool(field.flag, false, "")
			if err := cmd.Flags().Set(field.flag, "true"); err != nil {
				t.Fatalf("Set flag: %v", err)
			}

			input := &ClusterUpdateInput{}
			v := true
			field.set(input, &v)
			normalizeClusterUpdateInput(cmd, input)

			got := field.get(input)
			if got == nil || !*got {
				t.Fatalf("expected %s=true after explicit flag set", field.flag)
			}
		})
	}
}
