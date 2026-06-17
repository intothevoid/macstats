package collector

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
)

// Metrics holds all system metrics.
type Metrics struct {
	CPU struct {
		Usage  float64
		PerCPU []float64
	}
	GPU struct {
		Usage float64
	}
	System struct {
		Temperature float64
		MaxMemory   uint64
		UsedMemory  uint64
	}
}

func CollectMetrics() (Metrics, error) {
	cpu := parseCPU()
	perCPU := parsePerCoreCPU()
	gpu := parseGPU()
	temperature := parseTemperature()
	mem := parseMemory()
	cpu.PerCPU = perCPU
	return Metrics{
		CPU: cpu,
		GPU: gpu,
		System: struct{ Temperature float64; MaxMemory uint64; UsedMemory uint64 }{
			Temperature: temperature,
			MaxMemory:   mem.Max,
			UsedMemory:  mem.Used,
		},
	}, nil
}

func (m Metrics) String() string {
	data, _ := json.Marshal(m)
	return string(data)
}

func parseCPU() struct { Usage float64; PerCPU []float64 } {
	// top -l 1 -n 0 -F: "CPU usage: 1.58% user, 4.40% sys, 94.0% idle"
	out, err := exec.Command("top", "-l", "1", "-n", "0", "-F").Output()
	if err != nil {
		return struct{ Usage float64; PerCPU []float64 }{0, nil}
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "CPU usage:") {
			// Parse idle value (100.0 - idle = total CPU utilization)
			var idle float64
			for _, p := range strings.Split(line, ",") {
				trimmed := strings.TrimSpace(p)
				if strings.HasSuffix(trimmed, "% idle") {
					idle, _ = strconv.ParseFloat(trimmed[:len(trimmed)-5], 64)
				}
			}
			return struct{ Usage float64; PerCPU []float64 }{100.0 - idle, nil}
		}
	}
	return struct{ Usage float64; PerCPU []float64 }{0, nil}
}

func parsePerCoreCPU() []float64 {
	// sysctl hw.physicalcpu: total physical cores
	out, err := exec.Command("sysctl", "hw.physicalcpu").Output()
	if err != nil {
		return nil
	}
	trim := strings.TrimSpace(string(out))
	idx := strings.Index(trim, ":")
	if idx == -1 {
		return nil
	}
	n, _ := strconv.Atoi(strings.TrimSpace(trim[idx+1:]))
	// Return array of N core counts (just the count)
	cores := make([]float64, n)
	for i := 0; i < n; i++ {
		cores[i] = 1.0  // just represent N cores
	}
	return cores
}

func parseGPU() struct { Usage float64 } {
	// No GPU utilization available through standard CLI on Apple Silicon
	// Available only through IOKit/PMPwrMgmt C API
	return struct{ Usage float64 }{0}
}

func parseTemperature() float64 {
	// No CPU temperature data through standard CLI
	// Available only through IOKit
	return -1
}

func parseMemory() struct { Max uint64; Used uint64 } {
	out, err := exec.Command("sysctl", "hw.memsize").Output()
	if err != nil {
		return struct{ Max uint64; Used uint64 }{0, 0}
	}
	trim := strings.TrimSpace(string(out))
	idx := strings.Index(trim, ":")
	if idx == -1 {
		return struct{ Max uint64; Used uint64 }{0, 0}
	}
	max, _ := strconv.ParseUint(strings.TrimSpace(trim[idx+1:]), 10, 64)

	// vm_stat (pages, 16384 bytes per page on M-series)
	out, err = exec.Command("vm_stat").Output()
	if err != nil {
		return struct{ Max uint64; Used uint64 }{max, 0}
	}
	used := parseVMStat(out)
	return struct{ Max uint64; Used uint64 }{max, used}
}

func parseVMStat(data []byte) uint64 {
	pages := uint64(16384)  // 16KB pages
	var inUse uint64
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		idx := strings.Index(trimmed, ":")
		if idx == -1 {
			continue
		}
		val, _ := strconv.ParseUint(strings.TrimSpace(trimmed[idx+1:]), 10, 64)
		inUse += val * pages
	}
	return inUse
}
