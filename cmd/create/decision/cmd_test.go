package decision

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/accesstransparency/v1"
	"github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/output"
	. "github.com/openshift/rosa/pkg/test"
)

var _ = Describe("rosa attach policy", func() {
	Context("Create Command", func() {
		It("Returns Command", func() {

			cmd := NewCreateDecisionCommand()
			Expect(cmd).NotTo(BeNil())

			Expect(cmd.Use).To(Equal(use))
			Expect(cmd.Example).To(Equal(example))
			Expect(cmd.Short).To(Equal(short))
			Expect(cmd.Long).To(Equal(long))
			Expect(cmd.Args).NotTo(BeNil())
			Expect(cmd.Run).NotTo(BeNil())

			flag := cmd.Flags().Lookup("access-request")
			Expect(flag).NotTo(BeNil())
			flag = cmd.Flags().Lookup("decision")
			Expect(flag).NotTo(BeNil())
			flag = cmd.Flags().Lookup("justification")
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
				accessRequest: "fake-id",
			}
			output.SetOutput("")
		})

		AfterEach(func() {
			output.SetOutput("")
		})

		It("Returns an error if provide invalid decision type", func() {
			options.decision = "invalid-decision"
			runner := CreateDecisionRunner(options)
			err := runner(nil, t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf(
				"Invalid 'decision' value: '%s', should be one of '%s', '%s'",
				options.decision, v1.DecisionDecisionApproved, v1.DecisionDecisionDenied)))
		})

		It("Returns an error if decision is Denied without justification", func() {
			options.decision = "Denied"
			runner := CreateDecisionRunner(options)
			err := runner(nil, t.RosaRuntime, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf(
				"Non-empty value is required for 'justification' if 'decision' is set as '%s'",
				v1.DecisionDecisionDenied)))
		})

		It("Creates decision successfully", func() {
			options.decision = "Denied"
			options.justification = "mock-justification"
			decision, err := v1.NewDecision().ID("fake-decision-id").Decision(v1.DecisionDecisionDenied).Build()
			Expect(err).NotTo(HaveOccurred())
			t.ApiServer.AppendHandlers(
				testing.RespondWithJSON(
					http.StatusOK, FormatResource(decision)))
			runner := CreateDecisionRunner(options)
			t.StdOutReader.Record()
			err = runner(nil, t.RosaRuntime, nil, nil)
			Expect(err).NotTo(HaveOccurred())
			stdOut, _ := t.StdOutReader.Read()
			Expect(stdOut).To(Equal("INFO: Successfully created the decision for access request 'fake-id'\n"))
		})

	})
})
