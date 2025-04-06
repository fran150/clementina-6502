package computers

import (
	"github.com/fran150/clementina-6502/pkg/common"
)

type Computer interface {
	Tick(context *common.StepContext)
	Draw(context *common.StepContext)
}
