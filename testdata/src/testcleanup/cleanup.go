package testcleanup

func Setup() *Ctx { return &Ctx{} }

type Ctx struct{}

func (*Ctx) Teardown() {}
