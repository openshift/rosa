package breakglasscredential

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/breakglasscredential"
)

var _ = Describe("Break glass credential", func() {
	Context("AddBreakGlassCredentialFlags", func() {
		It("Should return the expected output", func() {
			cmd := makeCmd()
			args := breakglasscredential.AddBreakGlassCredentialFlags(cmd)
			Expect(args).To(Equal(breakGlassCredentialArgs))
		})
	})
})
