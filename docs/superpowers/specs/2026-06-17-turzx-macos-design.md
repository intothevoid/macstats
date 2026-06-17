# TURZX macOS Metrics Display Design

## Goal

Build a Go application for macOS on Apple Silicon that reads CPU, GPU, memory, and temperature metrics and displays them on a connected TURZX 3.5-inch USB screen. The primary implementation path is to reuse the vendor theme and serial protocol from the Windows package, with a custom renderer reserved as a contingency if protocol reuse proves insufficient.

## Scope

This design covers:

- metric collection on macOS Apple Silicon
- serial communication with the connected TURZX device
- reuse of selected vendor theme assets and payload structures
- isolation of reverse-engineered protocol logic from the rest of the app
- a curated `resources/` directory for committed assets

This design does not include:

- Windows support
- a fully custom framebuffer renderer as the primary path
- committing the full `35inchENG/` vendor directory into the repository

## Constraints

- Runtime target is macOS on Apple Silicon only.
- CGO or a helper binary is acceptable if needed for GPU usage and temperature metrics.
- The connected device is visible on the current machine as `/dev/cu.usbmodemUSB35INCHIPSV21`.
- The app should prefer existing vendor themes rather than requiring a new UI.
- `35inchENG/` should remain a local reverse-engineering source and should not be committed to GitHub.
- Only needed reusable assets should be copied into a new committed `resources/` tree.

## Architecture

The system should be split into four focused units:

### `internal/collector`

Responsible for collecting system metrics on macOS Apple Silicon. It should expose a stable Go data model for CPU, GPU, memory, and temperature values and should degrade gracefully when a metric is unavailable.

### `internal/protocol`

Responsible for understanding the TURZX serial protocol, theme payload structure, and metric update packets. This package should parse curated vendor-derived fixtures and construct outbound payloads without depending on metric collection details.

### `internal/display`

Responsible for opening the serial device, performing initialization writes, and sending theme or metric update packets produced by the protocol package. It should not know how metrics are collected or how payloads are derived internally.

### `cmd/macstats`

Responsible for configuration, startup sequencing, polling cadence, structured logging, and graceful shutdown. This command should wire together collector, protocol, and display components.

## Data Flow

At startup, the application should:

1. detect and open the TURZX serial device
2. load the selected curated theme payload from `resources/`
3. send the vendor-compatible initialization or theme payload required to wake the panel from its blank state
4. enter a timed refresh loop

During the refresh loop, the application should:

1. collect current metrics from the collector package
2. map those metrics into the vendor-compatible protocol fields
3. send incremental metric update packets when supported
4. fall back to larger payload writes if the protocol requires full refreshes

The initial polling interval should be `1s`. This is fast enough for a hardware stats display while keeping serial traffic and failure surface low during protocol validation.

## Metrics Strategy

The collector should provide:

- total CPU usage
- per-CPU or per-cluster values only if they are genuinely available and useful to the selected theme
- memory total and used values
- GPU usage when available
- temperature when available

Preferred sourcing order:

- CPU usage from native APIs such as `host_processor_info`, or a reliable command fallback during early bring-up
- memory from `sysctl`, `vm_stat`, or native APIs
- GPU usage and temperature from a helper binary or CGO bridge if shell-accessible sources are insufficient

When a metric is unavailable, the app should not fail the refresh loop. It should instead expose a sentinel or `N/A` representation that the protocol layer can map to a safe display value.

## Protocol Strategy

The protocol work should be staged to reduce hardware uncertainty:

1. verify the device can be opened and written to over serial on macOS
2. replay known vendor payloads to confirm the panel can be woken from a black screen
3. identify the minimal initialization bytes required for successful startup
4. determine which parts of a selected theme file are static assets versus dynamic metric fields
5. implement field-by-field metric updates, starting with CPU and memory
6. add GPU and temperature updates after the basic loop is proven

The protocol package should own:

- parsing of curated `.data` fixtures
- extraction of packet boundaries or headers
- mapping app metrics into protocol field updates
- construction of serial payloads for startup and refresh

## Asset Layout

The full `35inchENG/` directory should remain local and untracked as an external reference source. The committed repository should instead contain a curated `resources/` tree with only the files the app actually needs.

Planned structure:

- `resources/themes/` for selected vendor theme `.data` files
- `resources/images/` for any referenced reusable background or status bar assets needed by those themes
- `resources/testdata/` for reduced fixtures used by parser and protocol tests

Code must not hard-code paths into `35inchENG/`. Runtime and tests should load assets only from `resources/`.

## Error Handling

The application should fail fast only when the display cannot reasonably function:

- no compatible serial device is found
- the serial device cannot be opened
- the startup or theme initialization payload cannot be sent successfully

The application should degrade gracefully for:

- unavailable GPU metrics
- unavailable temperature metrics
- temporary collector errors on a single refresh cycle

In degraded cases, the app should continue updating the remaining metrics and log the failure context for diagnosis.

## Testing Strategy

Testing should be divided into unit-tested protocol logic and manual hardware validation:

### Automated tests

- unit tests for parsing curated `.data` fixtures
- unit tests for packet construction from known fixture inputs
- collector tests for output shape and fallback behavior on macOS

### Manual hardware checks

- confirm the screen wakes from black after startup payload transmission
- confirm a selected theme is displayed correctly
- confirm CPU and memory values change live
- confirm GPU and temperature values appear when those sources are implemented

This split ensures most logic is testable without hardware while acknowledging that final validation of the serial protocol requires the physical device.

## Contingency Path

If vendor theme reuse fails because the protocol is too opaque, the fallback is a custom framebuffer renderer built behind the same display abstraction. That contingency should not be the initial implementation target. The first milestone is successful theme-based startup and live metric updates on macOS.
