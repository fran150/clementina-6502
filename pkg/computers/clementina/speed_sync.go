package clementina

import (
	"math"

	"github.com/fran150/clementina-6502/pkg/core"
)

const hzPerMHz = 1_000_000

type miaSyncedSpeedController struct {
	core.SpeedController
	computer *ClementinaComputer
}

func newMiaSyncedSpeedController(computer *ClementinaComputer, controller core.SpeedController) core.SpeedController {
	synced := &miaSyncedSpeedController{
		SpeedController: controller,
		computer:        computer,
	}

	computer.SetMiaPhi2HzChangedHandler(func(hz uint32) {
		controller.SetTargetSpeed(hzToMHz(hz))
	})
	synced.syncMiaFromTarget()

	return synced
}

func (s *miaSyncedSpeedController) SpeedUp() {
	s.SpeedController.SpeedUp()
	s.syncMiaFromTarget()
}

func (s *miaSyncedSpeedController) SpeedDown() {
	s.SpeedController.SpeedDown()
	s.syncMiaFromTarget()
}

func (s *miaSyncedSpeedController) SetTargetSpeed(speedMhz float64) {
	s.SpeedController.SetTargetSpeed(speedMhz)
	s.syncMiaFromTarget()
}

func (s *miaSyncedSpeedController) syncMiaFromTarget() {
	s.computer.RequestMiaPhi2Hz(mhzToHz(s.SpeedController.GetTargetSpeed()))
}

func mhzToHz(speedMhz float64) uint32 {
	if speedMhz <= 0 {
		return 0
	}

	maxHz := ^uint32(0)
	if speedMhz >= float64(maxHz)/hzPerMHz {
		return maxHz
	}

	return uint32(math.Round(speedMhz * hzPerMHz))
}

func hzToMHz(hz uint32) float64 {
	return float64(hz) / hzPerMHz
}
