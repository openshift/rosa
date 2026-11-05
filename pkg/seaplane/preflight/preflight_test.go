// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package preflight

import (
	"context"
	"errors"
	"testing"

	"github.com/openshift/rosa/pkg/seaplane/pathbind"
)

func TestRunPreflightChecksAllPass(t *testing.T) {
	var order []string
	checks := []PreflightCheck{
		NewPreflightCheck("one", func(_ context.Context, _ *pathbind.ClusterCreateInput) error {
			order = append(order, "one")
			return nil
		}),
		NewPreflightCheck("two", func(_ context.Context, _ *pathbind.ClusterCreateInput) error {
			order = append(order, "two")
			return nil
		}),
	}

	results, err := RunPreflightChecks(context.Background(), checks, &pathbind.ClusterCreateInput{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, result := range results {
		if result.State != PreflightPass {
			t.Errorf("expected check %q to pass, got %q", result.Name, result.State)
		}
	}
	if len(order) != 2 || order[0] != "one" || order[1] != "two" {
		t.Errorf("checks did not run in order: %v", order)
	}
}

func TestRunPreflightChecksFailsFastOnError(t *testing.T) {
	sentinel := errors.New("boom")
	var ran []string
	checks := []PreflightCheck{
		NewPreflightCheck("first", func(_ context.Context, _ *pathbind.ClusterCreateInput) error {
			ran = append(ran, "first")
			return nil
		}),
		NewPreflightCheck("failing", func(_ context.Context, _ *pathbind.ClusterCreateInput) error {
			ran = append(ran, "failing")
			return sentinel
		}),
		NewPreflightCheck("never", func(_ context.Context, _ *pathbind.ClusterCreateInput) error {
			ran = append(ran, "never")
			return nil
		}),
	}

	results, err := RunPreflightChecks(context.Background(), checks, &pathbind.ClusterCreateInput{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected the sentinel error, got %v", err)
	}
	if len(ran) != 2 {
		t.Fatalf("expected fail-fast after the failing check, ran: %v", ran)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (pass + fail), got %d", len(results))
	}
	last := results[len(results)-1]
	if last.Name != "failing" || last.State != PreflightFail || last.Message != "boom" {
		t.Errorf("unexpected final result: %+v", last)
	}
	if results[0].State != PreflightPass {
		t.Errorf("expected the first result to pass, got %+v", results[0])
	}
}

func TestRunPreflightChecksEmptyRegistry(t *testing.T) {
	results, err := RunPreflightChecks(context.Background(), nil, &pathbind.ClusterCreateInput{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no results, got %d", len(results))
	}
}

func TestRunPreflightChecksMutatesInput(t *testing.T) {
	checks := []PreflightCheck{
		NewPreflightCheck("mutating", func(_ context.Context, input *pathbind.ClusterCreateInput) error {
			input.Name = "bridged-name"
			return nil
		}),
	}

	input := &pathbind.ClusterCreateInput{}
	if _, err := RunPreflightChecks(context.Background(), checks, input); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if input.Name != "bridged-name" {
		t.Errorf("expected check mutations to be visible to the caller, got %q", input.Name)
	}
}

func TestNewPreflightCheck(t *testing.T) {
	check := NewPreflightCheck("named", func(_ context.Context, _ *pathbind.ClusterCreateInput) error {
		return nil
	})
	if check.Name() != "named" {
		t.Errorf("expected name %q, got %q", "named", check.Name())
	}
	if err := check.Run(context.Background(), &pathbind.ClusterCreateInput{}); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
