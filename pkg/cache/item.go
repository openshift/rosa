// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package cache

import "time"

type Item struct {
	Object     interface{}
	Expiration time.Time
}

func (item Item) Expired() bool {
	if item.Expiration.IsZero() {
		return false
	}
	return time.Now().After(item.Expiration)
}
