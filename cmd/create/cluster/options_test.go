package cluster

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/openshift/rosa/pkg/test"
)

func TestOptions(t *testing.T) {
	cmd := exec.Command("rosa", "create", "cluster", "--help")
	out := bytes.Buffer{}
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to run the command: %v", err)
	}
	test.CompareWithFixture(t, out.Bytes(), test.WithExtension(".txt"))
}
