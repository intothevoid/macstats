package collector

import "testing"

func TestMetricsAvailabilityFlagsDefaultFalse(t *testing.T) {
	var metrics Metrics

	if metrics.CPU.Available {
		t.Fatal("expected CPU stats to default unavailable")
	}
	if metrics.GPU.Available {
		t.Fatal("expected GPU usage to default unavailable")
	}
	if metrics.System.TemperatureAvailable {
		t.Fatal("expected temperature to default unavailable")
	}
}

func TestParseVMStatReturnsZeroForGarbage(t *testing.T) {
	if got := parseVMStat([]byte("garbage: nope\nstill not vm_stat")); got != 0 {
		t.Fatalf("got %d, want 0", got)
	}
}

func TestMergeCPUStatsMarksAvailableWhenPerCoreStatsExist(t *testing.T) {
	cpu := mergeCPUStats(CPUStats{}, []float64{1, 1, 1, 1}, true)

	if !cpu.Available {
		t.Fatal("expected CPU stats to be available when per-core stats are available")
	}
	if len(cpu.PerCPU) != 4 {
		t.Fatalf("got %d per-core entries, want 4", len(cpu.PerCPU))
	}
}
