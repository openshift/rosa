package roles

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("generateClusterUpgradeInfo", func() {
	It("OK: Returns the cluster upgrade info string successfully", func() {
		info := generateClusterUpgradeInfo("cluster-key-01", "4.15.0", "auto")

		expected := "Account/Operator Role policies are not valid with upgrade version 4.15.0. " +
			"Run the following command(s) to upgrade the roles:\n" +
			"\trosa upgrade roles -c cluster-key-01 --cluster-version=4.15.0 --mode=auto\n\n" +
			", then run the upgrade command again:\n" +
			"\trosa upgrade cluster -c cluster-key-01\n"

		Expect(info).To(Equal(expected))
	})
})
