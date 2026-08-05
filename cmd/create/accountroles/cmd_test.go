package accountroles

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/interactive"
)

const cmdTestExternalID = "223B9588-36A5-ECA4-BE8D-7C673B77CEC1"

var _ = Describe("validateAccountRolesSTSExternalID", func() {
	It("accepts a valid external-id", func() {
		err := validateAccountRolesSTSExternalID(cmdTestExternalID)
		Expect(err).NotTo(HaveOccurred(), "valid external-id should pass validation")
	})

	It("accepts an empty external-id", func() {
		err := validateAccountRolesSTSExternalID("")
		Expect(err).NotTo(HaveOccurred(), "empty external-id should pass validation")
	})

	It("rejects an invalid external-id", func() {
		err := validateAccountRolesSTSExternalID("x")
		Expect(err).To(HaveOccurred(), "invalid external-id should fail validation")
	})
})

// OCP-43071 interactive flow (Prompter).
// Steps 6–7 already covered by cmd/verify/quota and cmd/verify/oc.
// Step 5 (login/token) deferred — login path, not account-roles prompts.
// Step 8 (invalid AWS credentials) deferred — fails in WithAWS()/client init
// before interactive prompts; needs a larger harness than this pilot.
var _ = Describe("OCP-43071 promptClassicAndHostedCP", func() {
	BeforeEach(func() {
		interactive.SetEnabled(true)
	})

	AfterEach(func() {
		interactive.SetEnabled(false)
	})

	It("asks classic (default true) then hosted-cp when no flags are set", func() {
		p := &recordingBoolPrompter{
			answers: map[string]bool{
				questionCreateClassic:  true,
				questionCreateHostedCP: false,
			},
		}

		createClassic, createHostedCP, _, _, err := promptClassicAndHostedCP(
			p, "classic help", "hosted help",
			false, false,
			false, false,
			false, false,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(p.asked).To(Equal([]string{questionCreateClassic, questionCreateHostedCP}))
		Expect(p.defaults[0]).To(Equal(true), "classic default should be Y/true")
		Expect(createClassic).To(BeTrue())
		Expect(createHostedCP).To(BeFalse())
	})

	It("skips classic question when --hosted-cp is set", func() {
		p := &recordingBoolPrompter{answers: map[string]bool{}}

		createClassic, createHostedCP, _, _, err := promptClassicAndHostedCP(
			p, "classic help", "hosted help",
			false, true, // createHostedCP from --hosted-cp
			false, true, // isHostedCPValueSet
			false, false,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(p.asked).To(BeEmpty(), "no classic/hosted prompts when --hosted-cp already set")
		Expect(createClassic).To(BeFalse())
		Expect(createHostedCP).To(BeTrue())
	})

	It("skips hosted-cp question when --classic flag is set", func() {
		p := &recordingBoolPrompter{answers: map[string]bool{}}

		createClassic, createHostedCP, _, _, err := promptClassicAndHostedCP(
			p, "classic help", "hosted help",
			true, false, // createClassic from --classic
			true, false, // isClassicValueSet
			true, false, // classicFlagChanged
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(p.asked).To(BeEmpty(), "no classic/hosted prompts when --classic already set")
		Expect(createClassic).To(BeTrue())
		Expect(createHostedCP).To(BeFalse())
	})

	It("uses flag-driven hosted-cp default when still prompting for hosted-cp after classic=false", func() {
		p := &recordingBoolPrompter{
			answers: map[string]bool{
				questionCreateClassic:  false,
				questionCreateHostedCP: true,
			},
		}

		_, createHostedCP, _, _, err := promptClassicAndHostedCP(
			p, "classic help", "hosted help",
			false, false,
			false, false,
			false, false,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(p.asked).To(Equal([]string{questionCreateClassic, questionCreateHostedCP}))
		Expect(p.defaults[1]).To(Equal(true), "hosted-cp default should be true when classic is false")
		Expect(createHostedCP).To(BeTrue())
	})
})
