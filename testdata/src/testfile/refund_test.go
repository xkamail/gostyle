package testfile

import "testing"

func TestRefundFee(t *testing.T) {
	_ = refundFee()
}

func TestCheckoutParams(t *testing.T) { // want `TestCheckoutParams uses nothing declared in refund.go; move it to checkout_test.go, next to checkout.go`
	_ = Params{}.ID
}
