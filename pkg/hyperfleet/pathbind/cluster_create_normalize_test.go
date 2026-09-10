package pathbind

import (
	"context"
	"testing"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/pathbind"
	"github.com/spf13/cobra"
)

func TestNormalizeClusterCreateInput_clearsUnsetSerializeImagePulls(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("serialize-image-pulls", false, "")

	input := &ClusterCreateInput{
		SerializeImagePulls: new(bool),
	}
	normalizeClusterCreateInput(cmd, input)

	if input.SerializeImagePulls != nil {
		t.Fatalf("expected nil SerializeImagePulls, got %v", *input.SerializeImagePulls)
	}

	obj := &v1alpha1.Cluster{}
	if err := pathbind.Expand(context.Background(), *input, obj); err != nil {
		t.Fatalf("Expand failed: %v", err)
	}
	cfg := obj.Spec.HostedCluster.Configuration
	if cfg != nil && cfg.Kubelet != nil && cfg.Kubelet.SerializeImagePulls != nil {
		t.Fatal("expected serializeImagePulls omitted from expanded cluster spec")
	}
}

func TestNormalizeClusterCreateInput_keepsChangedSerializeImagePulls(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("serialize-image-pulls", false, "")
	if err := cmd.Flags().Set("serialize-image-pulls", "true"); err != nil {
		t.Fatalf("Set flag: %v", err)
	}

	input := &ClusterCreateInput{
		SerializeImagePulls: new(bool),
	}
	*input.SerializeImagePulls = true
	normalizeClusterCreateInput(cmd, input)

	if input.SerializeImagePulls == nil || !*input.SerializeImagePulls {
		t.Fatal("expected SerializeImagePulls=true after explicit flag set")
	}
}

func TestNormalizeClusterUpdateInput_clearsUnsetSerializeImagePulls(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("serialize-image-pulls", false, "")

	input := &ClusterUpdateInput{
		SerializeImagePulls: new(bool),
	}
	normalizeClusterUpdateInput(cmd, input)

	if input.SerializeImagePulls != nil {
		t.Fatalf("expected nil SerializeImagePulls, got %v", *input.SerializeImagePulls)
	}
}
