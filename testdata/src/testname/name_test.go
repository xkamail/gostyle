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

func TestCheckout(t *testing.T) {}

func TestCheckout_WithCoupon(t *testing.T) {} // want `TestCheckout_WithCoupon: _ in a test name is only for TestType_Method; make "WithCoupon" a t.Run\("with coupon"\) case in TestCheckout`

func TestCheckout_coupon_expired(t *testing.T) {} // want `t.Run\("coupon expired"\) case in TestCheckout`

func TestItem_Price(t *testing.T) {}

func TestItem_Price_Zero(t *testing.T) {} // want `make "Zero" a t.Run\("zero"\) case in TestItem_Price`

func TestItem_Discount(t *testing.T) {} // want `t.Run\("discount"\) case in TestItem`

func TestRefund_Partial(t *testing.T) {} // want `t.Run\("partial"\) case in TestRefund`

func Test_lower(t *testing.T) {}
