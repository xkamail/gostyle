package testfile_test

import (
	"testing"

	"testfile"
)

func TestCheckout(t *testing.T) {
	_ = testfile.Checkout(testfile.Params{ID: 1})
}

func TestCheckoutViaHelper(t *testing.T) {
	doCheckout(t)
}

func TestRefund(t *testing.T) { // want `TestRefund uses nothing declared in checkout.go; move it to refund_test.go, next to refund.go`
	_ = testfile.Refund()
}

func TestNothing(t *testing.T) { // want `TestNothing uses nothing declared in checkout.go; move it to the test file of the source it tests`
}

func doCheckout(t *testing.T) {
	t.Helper()
	_ = testfile.Checkout(testfile.Params{})
}
