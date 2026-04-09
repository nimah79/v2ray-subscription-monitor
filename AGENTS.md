# Agent notes (Cursor / coding assistants)

Concise context for anyone (human or agent) changing this repository.

## Stack

- **Language:** Go (`go.mod` is authoritative for the Go version).
- **GUI:** `fyne.io/fyne/v2` (GLFW driver, **CGO required**) — expect **~20+ MiB** stripped; not fixable to ~5 MiB without dropping Fyne.
- **CLI:** `cmd/v2ray-subscription-cli` — **pure Go**, `CGO_ENABLED=0`, **~5–6 MiB** stripped; same fetch path as GUI.
- **Tray:** Fyne `desktop.App` + `fyne.io/systray` under the hood.
- **SVG tray asset:** `github.com/fyne-io/oksvg`, `github.com/srwiley/rasterx`.

## Important files

- **`main.go`** — All UI, prefs keys, polling (`doFetch`), systray refresh, quit path, warnings / fail-streak alerts.
- **`internal/subscription/fetch.go`** — GET + `Subscription-Userinfo` via `internal/userinfo`.
- **`internal/trayquit/`** — `TearDownSystrayForExternalLoop()` mirrors `systray.RunWithExternalLoop`’s end callback using `//go:linkname` to `fyne.io/systray.nativeEnd` plus `systray.Quit()`. Needed because Fyne’s GLFW driver may skip `trayStop` when `curWindow` is nil (e.g. window hidden to tray). **Fragile if systray renames `nativeEnd`.** Stub exists for non-desktop GOOS.
- **`internal/platform/`** — `tray_apply_darwin.c` + `tray_mode_darwin.go`: activation policy (Dock vs accessory). `SetTrayOnlyModeSync` for quit path. **Dock icon**: `EnsureNSApplication`, `SetDockIconFromPNG` (`dock_icon_*.c/go`), PNG squircle precompose (`dock_squircle_darwin.go`, 512× superellipse alpha). Stubs on non-Darwin.
- **`internal/trayicon/v2raystyle.go`** — Rasterize embedded SVG with foreground color; `InvalidateCache` on settings change.

## Conventions

- **Prefs:** Constants `pref*` in `main.go`; persist in `applyPrefs`, read with fallbacks where needed.
- **Threading:** Background work in goroutines; UI updates via `fyne.Do` / `runOnMain`.
- **Tray menu:** Built by `trayMenu()` — disabled title row, separator, Settings + Quit with theme icons.
- **No emoji** in tray title or summary labels; state uses **theme icons** (refresh / error / idle).

## Build

- **`make build`** — Release-oriented flags: `-buildvcs=false`, `-trimpath`, `-ldflags` (strip DWARF/symbol table + `-X main.appVersion=…`), `CGO_CFLAGS=-Os` for smaller native code, and on Darwin `-extldflags=-Wl,-no_warn_duplicate_libraries`. **`APP_VERSION`** defaults to the latest ancestor git tag (`v` prefix stripped) or `dev`; override when packaging. **Outputs** `v2ray-subscription-monitor-<APP_VERSION>` (and **`dist/`** artifacts use the same version infix).
- **`make build-debug`** — No strip; still injects **`main.appVersion`** via `-ldflags`.
- **`make build-cli` / `make dist-cli-*`** — Headless binary; `CGO_ENABLED=0`; cross-compile without a C toolchain.
- **`make dist-*`** (GUI) — Same strip/trim/VCS flags as `make build`, plus `CGO_CFLAGS=-Os`. Cross-builds need appropriate toolchains.

## When editing

- Prefer **small, task-scoped diffs**; match existing style (validators on entries, `runOnMain` for UI).
- Do **not** remove **`trayquit`** teardown from **`quitApplication`** without re-validating systray exit on macOS (hidden window → Quit).
- **`//go:linkname`** requires `_ "unsafe"` in `internal/trayquit/teardown.go`.
- Avoid new markdown files unless the user asks.

## Tests

- `go test ./...` — meaningful tests today live under `internal/userinfo`.
