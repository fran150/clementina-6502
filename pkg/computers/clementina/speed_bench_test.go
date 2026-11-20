package clementina

import "testing"

// BenchmarkAutomationCycles measures how many PHI2 cycles per second the host
// can emulate with the automation runner's per-cycle work, booting the kernel
// and running BASIC. Run with -benchtime=5000000x; MHz is the useful number.
func BenchmarkAutomationCycles(b *testing.B) {
	a, err := NewAutomationWithSDFolder("")
	if err != nil {
		b.Fatal(err)
	}
	defer a.Close()
	if _, err := a.Execute(AutomationRequest{Version: 1, Method: "reset"}); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	a.mu.Lock()
	for i := 0; i < b.N; i++ {
		a.tick()
	}
	a.mu.Unlock()
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds()/1e6, "MHz")
}
