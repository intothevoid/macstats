# TURZX macOS Metrics Display Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a macOS Apple Silicon Go app that wakes a connected TURZX 3.5-inch display, loads a curated vendor theme from `resources/`, and updates live system metrics over the device's serial protocol.

**Architecture:** The app is split into a macOS metrics collector, a TURZX protocol package that parses curated vendor fixtures and builds startup/update packets, a display transport that manages the serial device, and a small CLI that runs initialization plus a 1 Hz refresh loop. The first milestone is hardware bring-up using curated vendor theme payloads; metric field updates are layered on top after the device reliably displays a selected theme.

**Tech Stack:** Go 1.26, standard library, macOS system commands/native APIs, optional CGO/helper binary for advanced metrics, serial device access via `/dev/cu.*`.

---

### File Structure

**Create:**
- `cmd/macstats/main.go`
- `internal/protocol/theme.go`
- `internal/protocol/theme_test.go`
- `internal/protocol/update.go`
- `internal/display/serial.go`
- `internal/display/serial_test.go`
- `internal/collector/metrics_test.go`
- `resources/themes/3.5inchTheme1.data`
- `resources/testdata/theme_3_5inch_header.bin`
- `resources/README.md`
- `docs/superpowers/plans/2026-06-17-turzx-macos-implementation.md`

**Modify:**
- `internal/collector/metrics.go`
- `internal/display/send.go`
- `cmd/test/main.go`
- `.gitignore`

**Delete or retire later if superseded:**
- `internal/display/send.go` if `internal/display/serial.go` fully replaces it

### Task 1: Curate Resources And Lock Down Asset Boundaries

**Files:**
- Create: `resources/README.md`
- Create: `resources/themes/3.5inchTheme1.data`
- Create: `resources/testdata/theme_3_5inch_header.bin`
- Modify: `.gitignore`

- [ ] **Step 1: Write the failing asset-boundary check**

Create `resources/README.md` with the required contract so the repo has an explicit source-of-truth for assets:

```md
# Resources

This directory contains the only vendor-derived assets that the app may load at runtime or in tests.

- `themes/` contains curated `.data` theme files copied from the local `35inchENG/` reference bundle.
- `testdata/` contains reduced fixtures derived from curated themes for parser tests.

Rules:

1. Application code must not read from `35inchENG/`.
2. New vendor-derived assets must be copied here before use.
3. Only the minimum files needed by the selected theme should be committed.
```

- [ ] **Step 2: Run a repository check to verify it fails before curation**

Run: `rg -n "35inchENG/" .`
Expected: FAIL because current code still references `35inchENG/` directly, including `cmd/test/main.go`.

- [ ] **Step 3: Copy the selected theme and extract a small fixture**

Copy the chosen theme into the curated tree and create a reduced binary fixture from its leading bytes:

```bash
mkdir -p resources/themes resources/testdata
cp "35inchENG/config/3.5inchTheme1.data" "resources/themes/3.5inchTheme1.data"
dd if="resources/themes/3.5inchTheme1.data" of="resources/testdata/theme_3_5inch_header.bin" bs=1 count=512
```

- [ ] **Step 4: Update `.gitignore` so the vendor dump stays local**

Add these lines to `.gitignore`:

```gitignore
35inchENG/
```

- [ ] **Step 5: Re-run the repository check and verify only curated paths remain**

Run: `rg -n "35inchENG/" .`
Expected: PASS only for documentation comments that explicitly describe the local reference bundle, and no runtime code paths should depend on it.

- [ ] **Step 6: Commit**

```bash
git add .gitignore resources/README.md resources/themes/3.5inchTheme1.data resources/testdata/theme_3_5inch_header.bin
git commit -m "chore: curate TURZX resources"
```

### Task 2: Replace The Placeholder Serial Writer With A Real Display Transport

**Files:**
- Create: `internal/display/serial.go`
- Create: `internal/display/serial_test.go`
- Modify: `internal/display/send.go`
- Modify: `cmd/test/main.go`

- [ ] **Step 1: Write the failing transport tests**

Create `internal/display/serial_test.go`:

```go
package display

import "testing"

func TestNormalizeDevicePathRejectsEmpty(t *testing.T) {
	if _, err := normalizeDevicePath(""); err == nil {
		t.Fatal("expected error for empty device path")
	}
}

func TestBuildOpenCandidatesIncludesExplicitPath(t *testing.T) {
	got := buildOpenCandidates("/dev/cu.usbmodemUSB35INCHIPSV21")
	if len(got) == 0 || got[0] != "/dev/cu.usbmodemUSB35INCHIPSV21" {
		t.Fatalf("unexpected candidates: %#v", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/display -run 'TestNormalizeDevicePathRejectsEmpty|TestBuildOpenCandidatesIncludesExplicitPath'`
Expected: FAIL with undefined `normalizeDevicePath` and `buildOpenCandidates`.

- [ ] **Step 3: Write the minimal serial transport**

Create `internal/display/serial.go`:

```go
package display

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type SerialWriter struct {
	devicePath string
}

func NewSerialWriter(devicePath string) (*SerialWriter, error) {
	path, err := normalizeDevicePath(devicePath)
	if err != nil {
		return nil, err
	}
	return &SerialWriter{devicePath: path}, nil
}

func normalizeDevicePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("device path is required")
	}
	return path, nil
}

func buildOpenCandidates(path string) []string {
	return []string{path}
}

func DetectUSBSerialDevices() ([]string, error) {
	return filepath.Glob("/dev/cu.usbmodem*")
}

func (w *SerialWriter) Write(payload []byte) error {
	fd, err := os.OpenFile(w.devicePath, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer fd.Close()
	_, err = fd.Write(payload)
	return err
}
```

- [ ] **Step 4: Adapt the existing writer wrapper**

Replace `internal/display/send.go` with:

```go
package display

type Writer struct {
	serial *SerialWriter
}

func NewWriter(device string) (*Writer, error) {
	serial, err := NewSerialWriter(device)
	if err != nil {
		return nil, err
	}
	return &Writer{serial: serial}, nil
}

func (w *Writer) WriteFrame(buffer []byte) error {
	return w.serial.Write(buffer)
}
```

- [ ] **Step 5: Update the bring-up test command to use curated resources**

Refactor `cmd/test/main.go` to read `resources/themes/3.5inchTheme1.data` and use the new writer constructor:

```go
dev := "/dev/cu.usbmodemUSB35INCHIPSV21"
dataFile := "resources/themes/3.5inchTheme1.data"
writer, err := display.NewWriter(dev)
if err != nil {
	log.Fatal(err)
}
if err := writer.WriteFrame(data); err != nil {
	log.Fatal(err)
}
```

- [ ] **Step 6: Run tests to verify the transport passes**

Run: `go test ./internal/display`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/display/serial.go internal/display/serial_test.go internal/display/send.go cmd/test/main.go
git commit -m "feat: add TURZX serial transport"
```

### Task 3: Parse Curated Theme Files And Expose Startup Payloads

**Files:**
- Create: `internal/protocol/theme.go`
- Create: `internal/protocol/theme_test.go`
- Create: `resources/testdata/theme_3_5inch_header.bin`

- [ ] **Step 1: Write the failing parser tests**

Create `internal/protocol/theme_test.go`:

```go
package protocol

import (
	"os"
	"testing"
)

func TestParseThemeSeparatesDotNetHeaderFromPayload(t *testing.T) {
	data, err := os.ReadFile("../../resources/themes/3.5inchTheme1.data")
	if err != nil {
		t.Fatal(err)
	}
	theme, err := ParseTheme(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(theme.Payload) == 0 {
		t.Fatal("expected payload bytes")
	}
	if theme.HeaderLength != 256 {
		t.Fatalf("got header length %d", theme.HeaderLength)
	}
}

func TestParseThemeRejectsShortBuffers(t *testing.T) {
	if _, err := ParseTheme([]byte{0x01, 0x02}); err == nil {
		t.Fatal("expected short buffer error")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/protocol -run 'TestParseThemeSeparatesDotNetHeaderFromPayload|TestParseThemeRejectsShortBuffers'`
Expected: FAIL with undefined `ParseTheme`.

- [ ] **Step 3: Implement the minimal theme parser**

Create `internal/protocol/theme.go`:

```go
package protocol

import "errors"

type Theme struct {
	Raw          []byte
	Header       []byte
	Payload      []byte
	HeaderLength int
}

func ParseTheme(data []byte) (Theme, error) {
	if len(data) <= 256 {
		return Theme{}, errors.New("theme buffer too short")
	}
	return Theme{
		Raw:          data,
		Header:       data[:256],
		Payload:      data[256:],
		HeaderLength: 256,
	}, nil
}
```

- [ ] **Step 4: Add a startup payload helper**

Extend `internal/protocol/theme.go` with:

```go
func (t Theme) StartupPayload() []byte {
	return t.Raw
}
```

- [ ] **Step 5: Run the protocol tests**

Run: `go test ./internal/protocol`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/protocol/theme.go internal/protocol/theme_test.go resources/testdata/theme_3_5inch_header.bin
git commit -m "feat: parse curated TURZX themes"
```

### Task 4: Build A Safer macOS Collector API With Explicit Degradation

**Files:**
- Modify: `internal/collector/metrics.go`
- Create: `internal/collector/metrics_test.go`

- [ ] **Step 1: Write the failing collector tests**

Create `internal/collector/metrics_test.go`:

```go
package collector

import "testing"

func TestMetricsAvailabilityFlagsDefaultToFalse(t *testing.T) {
	var metrics Metrics
	if metrics.GPU.Available {
		t.Fatal("expected GPU availability to default false")
	}
	if metrics.System.TemperatureAvailable {
		t.Fatal("expected temperature availability to default false")
	}
}

func TestParseVMStatRejectsGarbage(t *testing.T) {
	if got := parseVMStat([]byte("nonsense")); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/collector -run 'TestMetricsAvailabilityFlagsDefaultToFalse|TestParseVMStatRejectsGarbage'`
Expected: FAIL because `Available` and `TemperatureAvailable` fields do not exist.

- [ ] **Step 3: Update the collector data model**

Refactor `internal/collector/metrics.go` so the exported types are stable and explicit:

```go
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

type Metrics struct {
	CPU    CPUStats
	GPU    GPUStats
	System SystemStats
}
```

- [ ] **Step 4: Make `CollectMetrics` degrade explicitly**

Update `CollectMetrics` to set availability flags instead of using magic numbers:

```go
func CollectMetrics() (Metrics, error) {
	cpu := parseCPU()
	cpu.PerCPU = parsePerCoreCPU()
	mem := parseMemory()

	return Metrics{
		CPU: CPUStats{
			Usage:     cpu.Usage,
			PerCPU:    cpu.PerCPU,
			Available: true,
		},
		GPU: GPUStats{
			Usage:     0,
			Available: false,
		},
		System: SystemStats{
			Temperature:          0,
			TemperatureAvailable: false,
			MaxMemory:            mem.Max,
			UsedMemory:           mem.Used,
		},
	}, nil
}
```

- [ ] **Step 5: Run the collector tests**

Run: `go test ./internal/collector`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/collector/metrics.go internal/collector/metrics_test.go
git commit -m "refactor: make collector degradation explicit"
```

### Task 5: Add A Minimal macOS CLI That Loads A Theme And Replays It

**Files:**
- Create: `cmd/macstats/main.go`
- Modify: `internal/protocol/theme.go`
- Modify: `internal/display/serial.go`

- [ ] **Step 1: Write the failing CLI smoke test**

Add this test to `internal/protocol/theme_test.go`:

```go
func TestThemeStartupPayloadReturnsRawBytes(t *testing.T) {
	theme := Theme{Raw: []byte{0x01, 0x02, 0x03}}
	if got := theme.StartupPayload(); len(got) != 3 {
		t.Fatalf("unexpected payload length: %d", len(got))
	}
}
```

- [ ] **Step 2: Run the targeted tests**

Run: `go test ./internal/protocol -run TestThemeStartupPayloadReturnsRawBytes`
Expected: PASS if Task 3 was completed; otherwise fix that first.

- [ ] **Step 3: Implement the CLI**

Create `cmd/macstats/main.go`:

```go
package main

import (
	"log"
	"os"

	"macstats/internal/display"
	"macstats/internal/protocol"
)

func main() {
	device := "/dev/cu.usbmodemUSB35INCHIPSV21"
	if env := os.Getenv("MACSTATS_DEVICE"); env != "" {
		device = env
	}

	data, err := os.ReadFile("resources/themes/3.5inchTheme1.data")
	if err != nil {
		log.Fatal(err)
	}

	theme, err := protocol.ParseTheme(data)
	if err != nil {
		log.Fatal(err)
	}

	writer, err := display.NewWriter(device)
	if err != nil {
		log.Fatal(err)
	}

	if err := writer.WriteFrame(theme.StartupPayload()); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 4: Run formatting and tests**

Run: `gofmt -w cmd/macstats/main.go internal/collector/metrics.go internal/collector/metrics_test.go internal/display/serial.go internal/display/serial_test.go internal/display/send.go internal/protocol/theme.go internal/protocol/theme_test.go cmd/test/main.go`
Expected: files rewritten without error

Run: `go test ./...`
Expected: PASS

- [ ] **Step 5: Run the hardware bring-up command**

Run: `go run ./cmd/macstats`
Expected: the TURZX screen should leave the black state and display the selected theme, or the command should fail with a precise open/write/parsing error to debug next.

- [ ] **Step 6: Commit**

```bash
git add cmd/macstats/main.go internal/protocol/theme.go internal/protocol/theme_test.go
git commit -m "feat: replay curated TURZX theme on macOS"
```

### Task 6: Add Incremental Metric Updates For CPU And Memory

**Files:**
- Create: `internal/protocol/update.go`
- Modify: `cmd/macstats/main.go`
- Modify: `internal/protocol/theme_test.go`

- [ ] **Step 1: Write the failing update-construction tests**

Append to `internal/protocol/theme_test.go`:

```go
func TestBuildMetricSnapshotIncludesCPUAndMemory(t *testing.T) {
	payload := BuildMetricSnapshot(12.5, 4096, 8192)
	if len(payload) == 0 {
		t.Fatal("expected non-empty metric payload")
	}
}
```

- [ ] **Step 2: Run the targeted tests to verify failure**

Run: `go test ./internal/protocol -run TestBuildMetricSnapshotIncludesCPUAndMemory`
Expected: FAIL with undefined `BuildMetricSnapshot`.

- [ ] **Step 3: Implement a temporary metric snapshot payload builder**

Create `internal/protocol/update.go`:

```go
package protocol

import "fmt"

func BuildMetricSnapshot(cpuUsage float64, usedMemory uint64, maxMemory uint64) []byte {
	return []byte(fmt.Sprintf("CPU=%.1f;MEM=%d/%d", cpuUsage, usedMemory, maxMemory))
}
```

- [ ] **Step 4: Extend the CLI refresh loop**

Update `cmd/macstats/main.go` after the startup write:

```go
	metrics, err := collector.CollectMetrics()
	if err != nil {
		log.Fatal(err)
	}

	update := protocol.BuildMetricSnapshot(
		metrics.CPU.Usage,
		metrics.System.UsedMemory,
		metrics.System.MaxMemory,
	)
	if err := writer.WriteFrame(update); err != nil {
		log.Fatal(err)
	}
```

This is intentionally temporary. Replace the string payload with a real vendor-compatible field update once protocol boundaries are identified from device behavior.

- [ ] **Step 5: Run the full test suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 6: Run manual hardware validation**

Run: `go run ./cmd/macstats`
Expected: either the theme remains stable and accepts the update packet, or the device rejects/ignores it, which gives the next reverse-engineering target.

- [ ] **Step 7: Commit**

```bash
git add internal/protocol/update.go cmd/macstats/main.go internal/protocol/theme_test.go
git commit -m "feat: add initial CPU and memory update loop"
```

### Task 7: Document Reverse-Engineering Findings And Next Metrics

**Files:**
- Modify: `docs/superpowers/specs/2026-06-17-turzx-macos-design.md`
- Create: `docs/turzx-protocol-notes.md`

- [ ] **Step 1: Write a protocol notes document**

Create `docs/turzx-protocol-notes.md` with:

```md
# TURZX Protocol Notes

## Confirmed

- Device path observed on macOS:
- Theme file selected:
- Theme startup payload behavior:
- Known header length:

## Unknown

- Dynamic field offsets:
- CPU field encoding:
- Memory field encoding:
- GPU field encoding:
- Temperature field encoding:

## Next experiments

1. Compare multiple `.data` theme files to identify variable regions.
2. Capture device behavior for startup-only versus startup-plus-update writes.
3. Add GPU and temperature sources after CPU and memory field updates are confirmed.
```

- [ ] **Step 2: Run a quick documentation review**

Run: `sed -n '1,220p' docs/turzx-protocol-notes.md`
Expected: all placeholders either filled with current findings or intentionally left in the `Unknown` section only.

- [ ] **Step 3: Commit**

```bash
git add docs/turzx-protocol-notes.md docs/superpowers/specs/2026-06-17-turzx-macos-design.md
git commit -m "docs: record TURZX protocol findings"
```
