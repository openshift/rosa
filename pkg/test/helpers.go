package test

import (
	"io"
	"os"

	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
)

func RunWithOutputCapture(runWithRuntime func(*rosa.Runtime, *cobra.Command) error,
	runtime *rosa.Runtime, cmd *cobra.Command) (string, string, error) {
	var err error
	var stdout []byte
	var stderr []byte

	rout, wout, _ := os.Pipe()
	tmpout := os.Stdout
	rerr, werr, _ := os.Pipe()
	tmperr := os.Stderr
	defer func() {
		os.Stdout = tmpout
		os.Stderr = tmperr
	}()
	os.Stdout = wout
	os.Stderr = werr

	go func() {
		err = runWithRuntime(runtime, cmd)
		wout.Close()
		werr.Close()
	}()
	stdout, _ = io.ReadAll(rout)
	stderr, _ = io.ReadAll(rerr)

	return string(stdout), string(stderr), err
}
