package accessrequest

import (
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/accesstransparency/v1"
	"github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/output"
	. "github.com/openshift/rosa/pkg/test"
)

var _ = Describe("rosa describe accessrequest", func() {
	Context("Create Command", func() {
		It("Returns Command", func() {

			cmd := NewDescribeAccessRequestCommand()
			Expect(cmd).NotTo(BeNil())

			Expect(cmd.Use).To(Equal(use))
			Expect(cmd.Example).To(Equal(example))
			Expect(cmd.Short).To(Equal(short))
			Expect(cmd.Long).To(Equal(long))
			Expect(cmd.Args).NotTo(BeNil())
			Expect(cmd.Run).NotTo(BeNil())

			flag := cmd.Flags().Lookup("id")
			Expect(flag).NotTo(BeNil())
		})
	})

	Context("Execute command", func() {
		var (
			t       *TestingRuntime
			options *Options
		)

		BeforeEach(func() {
			t = NewTestRuntime()
			options = &Options{
				id: "mock-id",
			}
			output.SetOutput("")
		})

		AfterEach(func() {
			output.SetOutput("")
		})

		It("Returns an error if access request not found", func() {
			t.ApiServer.AppendHandlers(
				testing.RespondWithJSON(
					http.StatusNotFound, ""))
			runner := DescribeAccessRequestRunner(options)
			err := runner(nil, t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("The Access Request with id 'mock-id' does not exist"))
		})

		It("Describe access request successfully", func() {
			layout := "Jan 2, 2006 at 3:04pm (MST)"
			mock_time, _ := time.Parse(layout, "Sep 28, 2024 at 8:35pm (PST)")
			decision := v1.NewDecision().
				Decision(v1.DecisionDecisionApproved).
				DecidedBy("mock-decider").
				CreatedAt(mock_time).
				Justification("mock-decision-justification")
			accessRequest, err := v1.NewAccessRequest().
				ID("mock-id").
				SubscriptionId("mock-subscription-id").
				ClusterId("mock-cluster-id").
				CreatedAt(mock_time).
				DeadlineAt(mock_time).
				Duration("8h").
				RequestedBy("mock-requestor").
				Justification("mock-justification").
				SupportCaseId("mock-case-id").
				Status(v1.NewAccessRequestStatus().State(v1.AccessRequestStateApproved)).
				Decisions(decision).
				Build()
			Expect(err).NotTo(HaveOccurred())
			t.ApiServer.AppendHandlers(
				testing.RespondWithJSON(
					http.StatusOK, FormatResource(accessRequest)))
			runner := DescribeAccessRequestRunner(options)
			t.StdOutReader.Record()
			err = runner(nil, t.RosaRuntime, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal(`
ID:                                mock-id
Subscription ID:                   mock-subscription-id
Cluster ID:                        mock-cluster-id
Support Case ID:                   mock-case-id
Requested By:                      mock-requestor
Created At:                        Sat Sep 28 20:35:00 UTC 2024
Respond By:                        Sat Sep 28 20:35:00 UTC 2024
Request Duration:                  8h
Justification:                     mock-justification
Status:                            Approved
Decisions:                           
  - Decision:                      Approved
    Decided By:                    mock-decider
    Created At:                    Sat Sep 28 20:35:00 UTC 2024
    Justification:                 mock-decision-justification
`))
		})

	})
})
