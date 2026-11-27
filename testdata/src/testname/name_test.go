package testname

import "testing"

func TestName(t *testing.T) {
	t.Run("not found", func(t *testing.T) {})
	t.Run("ok", func(t *testing.T) {})
	t.Run("OTP expired", func(t *testing.T) {})
	t.Run("active account should not have deletedAt", func(t *testing.T) {})
	t.Run("live_item_succeeds", func(t *testing.T) {}) // want `subtest name "live_item_succeeds": write lowercase words with spaces, e.g. "live item succeeds"`
	t.Run("getItem", func(t *testing.T) {})            // want `subtest name "getItem": write lowercase words with spaces, e.g. "get item"`
	t.Run("WithCoupon", func(t *testing.T) {})         // want `e.g. "with coupon"`
	t.Run("GET /items", func(t *testing.T) {})         // want `subtest name "GET /items" has a /`
	name := "case_one"
	t.Run(name, func(t *testing.T) {})
}

func BenchmarkName(b *testing.B) {
	b.Run("snake_case", func(b *testing.B) {})
}
