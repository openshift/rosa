package decision

import (
	"context"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	v1 "github.com/openshift-online/ocm-sdk-go/accesstransparency/v1"
	"github.com/spf13/cobra"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "decision"
	short   = "Create a decision for an access request"
	long    = short
	example = `  # Create a decision for an access request to approve it
  rosa create decision --access-request <access_request_id> --decision Approved
  `
)

type Options struct {
	accessRequest string
	decision      string
	justification string
}

func NewDecisionOptions() *Options {
	return &Options{}
}

func NewCreateDecisionCommand() *cobra.Command {

	options := NewDecisionOptions()
	cmd := &cobra.Command{
		Use:     use,
		Aliases: []string{"access-request"},
		Short:   short,
		Long:    long,
		Example: example,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), CreateDecisionRunner(options)),
		Args:    cobra.NoArgs,
	}

	flags := cmd.Flags()
	flags.StringVarP(
		&options.accessRequest,
		"access-request",
		"a",
		"",
		"ID of the Access Request to add decision (required).",
	)
	flags.StringVarP(
		&options.decision,
		"decision",
		"d",
		"",
		"Decision created for the access request, valid values are 'Approved' or 'Denied' (required).",
	)
	flags.StringVarP(
		&options.justification,
		"justification",
		"j",
		"",
		"Justification for the decision, required if decision is 'Denied'.",
	)
	cmd.MarkFlagRequired("access-request")
	cmd.MarkFlagRequired("decision")
	return cmd
}

func CreateDecisionRunner(options *Options) rosa.CommandRunner {
	return func(_ context.Context, r *rosa.Runtime, _ *cobra.Command, _ []string) error {
		err := ValidateDecisionOptions(options)
		if err != nil {
			return err
		}
		err = r.OCMClient.CreateDecision(options.accessRequest, options.decision, options.justification)
		if err != nil {
			return err
		} else {
			r.Reporter.Infof("Successfully created the decision for access request '%s'", options.accessRequest)
			return nil
		}
	}
}

func ValidateDecisionOptions(options *Options) error {
	decisionStr := cases.Title(language.English, cases.Compact).String(options.decision)
	switch v1.DecisionDecision(decisionStr) {
	case v1.DecisionDecisionDenied:
		if strings.TrimSpace(options.justification) == "" {
			return errors.Errorf("Non-empty value is required for 'justification' if 'decision' is set as '%s'",
				v1.DecisionDecisionDenied)
		}
		return nil
	case v1.DecisionDecisionApproved:
		return nil
	default:
		return errors.Errorf("Invalid 'decision' value: '%s', should be one of '%s', '%s'",
			options.decision, v1.DecisionDecisionApproved, v1.DecisionDecisionDenied)
	}
}
