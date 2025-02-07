package computers

import "github.com/fran150/clementina6502/pkg/components/common"

type Computer interface {
	Step(context *common.StepContext)
	UpdateDisplay(context *common.StepContext)
	RunEventLoop()
}
