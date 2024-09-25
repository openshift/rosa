package ocm

import v1 "github.com/openshift-online/ocm-sdk-go/accesstransparency/v1"

func (c *Client) CreateDecision(accessRequest string, decision string, justification string) error {
	decisionSpec, err := v1.NewDecision().
		Decision(v1.DecisionDecision(decision)).
		Justification(justification).
		Build()
	if err != nil {
		return err
	}
	_, err = c.ocm.AccessTransparency().V1().AccessRequests().
		AccessRequest(accessRequest).Decisions().
		Add().
		Body(decisionSpec).
		Send()
	if err != nil {
		return err
	}
	return nil
}
