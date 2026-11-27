package testname_test

import (
	"testing"

	"testname"
)

func TestItem_Price(t *testing.T) {
	_ = testname.Item{}.Price()
}

func TestCheckout_Guest(t *testing.T) { // want `t.Run\("guest"\) case in TestCheckout`
	_ = testname.Checkout()
}
