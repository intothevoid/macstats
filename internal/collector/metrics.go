package collector

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
)

type CPUStats struct {
	Usage     float64
	PerCPU    []float64
	Available bool
}

type GPUStats struct {
	Usage     float64
	Available bool
}

type SystemStats struct {
	Temperature          float64
	TemperatureAvailable bool
	MaxMemory            uint64
	UsedMemory           uint64
}

// Metrics holds all system metrics.
type Metrics struct {
	CPU    CPUStats
	GPU    GPUStats
	System SystemStats
}

func CollectMetrics() (Metrics, error) {
	cpu := parseCPU()
	perCPU, perCPUAvailable := parsePerCoreCPU()
	gpu := parseGPU()
	temperature, temperatureAvailable := parseTemperature()
	mem := parseMemory()
	cpu = mergeCPUStats(cpu, perCPU, perCPUAvailable)
	return Metrics{
		CPU: cpu,
		GPU: gpu,
		System: SystemStats{
			Temperature:          temperature,
			TemperatureAvailable: temperatureAvailable,
			MaxMemory:            mem.MaxMemory,
			UsedMemory:           mem.UsedMemory,
		},
	}, nil
}

func mergeCPUStats(cpu CPUStats, perCPU []float64, perCPUAvailable bool) CPUStats {
	cpu.PerCPU = perCPU
	cpu.Available = cpu.Available || perCPUAvailable
	return cpu
}

func (m Metrics) String() string {
	data, _ := json.Marshal(m)
	return string(data)
}

func parseCPU() CPUStats {
	// top -l 1 -n 0 -F: "CPU usage: 1.58% user, 4.40% sys, 94.0% idle"
	out, err := exec.Command("top", "-l", "1", "-n", "0", "-F").Output()
	if err != nil {
		return CPUStats{}
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
			return CPUStats{
				Usage:     100.0 - idle,
				Available: true,
			}
		}
	}
	return CPUStats{}
}

func parsePerCoreCPU() ([]float64, bool) {
	// sysctl hw.physicalcpu: total physical cores
	out, err := exec.Command("sysctl", "hw.physicalcpu").Output()
	if err != nil {
		return nil, false
	}
	trim := strings.TrimSpace(string(out))
	idx := strings.Index(trim, ":")
	if idx == -1 {
		return nil, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(trim[idx+1:]))
	if err != nil {
		return nil, false
	}
	// Return array of N core counts (just the count)
	cores := make([]float64, n)
	for i := 0; i < n; i++ {
		cores[i] = 1.0 // just represent N cores
	}
	return cores, true
}

func parseGPU() GPUStats {
	// No GPU utilization available through standard CLI on Apple Silicon
	// Available only through IOKit/PMPwrMgmt C API
	return GPUStats{}
}

func parseTemperature() (float64, bool) {
	// No CPU temperature data through standard CLI
	// Available only through IOKit
	return 0, false
}

func parseMemory() SystemStats {
	out, err := exec.Command("sysctl", "hw.memsize").Output()
	if err != nil {
		return SystemStats{}
	}
	trim := strings.TrimSpace(string(out))
	idx := strings.Index(trim, ":")
	if idx == -1 {
		return SystemStats{}
	}
	max, _ := strconv.ParseUint(strings.TrimSpace(trim[idx+1:]), 10, 64)

	// vm_stat (pages, 16384 bytes per page on M-series)
	out, err = exec.Command("vm_stat").Output()
	if err != nil {
		return SystemStats{MaxMemory: max}
	}
	used := parseVMStat(out)
	return SystemStats{
		MaxMemory:  max,
		UsedMemory: used,
	}
}

func parseVMStat(data []byte) uint64 {
	pages := uint64(16384) // 16KB pages
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
