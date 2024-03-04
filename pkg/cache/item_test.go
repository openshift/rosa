package cache

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Item", func() {
	It("should return false for an item with zero expiration time", func() {
		item := Item{
			Object:     "test",
			Expiration: time.Time{},
		}
		Expect(item.Expired()).To(BeFalse())
	})

	It("should return false for an item with expiration time in the future", func() {
		futureExpiration := time.Now().Add(1 * time.Hour)
		item := Item{
			Object:     "test",
			Expiration: futureExpiration,
		}
		Expect(item.Expired()).To(BeFalse())
	})

	It("should return true for an item with expiration time in the past", func() {
		pastExpiration := time.Now().Add(-1 * time.Hour)
		item := Item{
			Object:     "test",
			Expiration: pastExpiration,
		}
		Expect(item.Expired()).To(BeTrue())
	})
})
