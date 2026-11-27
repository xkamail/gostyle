package testname

type Item struct{}

func (Item) Price() int { return 0 }

func Checkout() error { return nil }
