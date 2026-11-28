package stutter

type Stutter struct{}

type StutterConfig struct{} // want `StutterConfig stutters as stutter.StutterConfig; name it Config`

type Stutters struct{}

type stutterHelper struct{}

const StutterV2 = 1 // want `StutterV2 stutters as stutter.StutterV2; name it V2`

func NewStutter() *Stutter { return nil } // want `NewStutter stutters as stutter.NewStutter; name it New`

func (Stutter) StutterName() {}
