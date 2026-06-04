# CLAUDE.md — neotrinkey

Viam module for the Adafruit NeoTrinkey (SAMD21 USB key, 4 NeoPixels). Registry namespace
`viam` (org `viam-dev`). Built as the LED for the `light-painting` module.

## Models (all in one binary; `cmd/module/main.go` registers all three)

- `viam:neotrinkey:trinkey` (`rdk:component:generic`, `trinkey/real.go`) — real driver over
  USB serial via `go.bug.st/serial`. Writes `r,g,b,…\n` lines to the device.
- `viam:neotrinkey:trinkey-sim` (`rdk:component:generic`, `trinkey/sim.go`) — in-memory; logs.
- `viam:neotrinkey:trinkey-scene` (`rdk:service:world_state_store`, `trinkey/scene.go`) —
  4-sphere visualizer (size = intensity, color = RGB), built on `viam-viz-helpers-go`.

`trinkey/pixels.go` holds the shared driver core (DoCommand + pixel state + brightness); the
`writer` interface is the only difference between sim and real. The driver pushes LED state
to the visualizer **in-process** (`scene.Lookup`/`visuals.Register`) — both ship in one binary.

## DoCommand

`set_pixels` `{pixels:[[r,g,b],…]}`, `set_all`/`set_color` `{color:[r,g,b]}`, `off`,
`brightness` `{brightness:0..1}`, `get_pixels`. Brightness scales the hardware output but the
reported state stays raw.

## Firmware

`firmware/boot.py` enables `usb_cdc.data`; `firmware/code.py` parses the line protocol and
drives `board.NEOPIXEL` (4 px). Copy both to the CIRCUITPY drive. `serial_path` is the
device's **data** CDC port (the second ttyACM/usbmodem it exposes).

## Build / test / deploy

```bash
make test
make build-all   # all 5 platforms -> dist/*.tar.gz (pure Go, cross-compiles)
viam module update
viam module upload --version X.Y.Z --platform <os/arch> --upload dist/neotrinkey-<os>-<arch>.tar.gz
```

## Conventions / gotchas

- **RDK v0.124.0**, Go 1.25. Generic *component* API is `go.viam.com/rdk/components/generic`
  (NOT `services/generic`). `resource.Named` supplies Name/Status/DoCommand-default;
  `resource.AlwaysRebuild` supplies Reconfigure; define `Close` (and `DoCommand`) yourself.
- The visualizer must define `DoCommand` explicitly to disambiguate `SceneServiceBase` from
  `resource.Named` (same gotcha as light-painting's painting-scene).
- All-arch builds depend on `go.bug.st/serial` being **pure Go** (no cgo) — keep it that way.
- The tar entrypoint is `bin/neotrinkey`; `build-all` puts the binary at `bin/` inside each
  tar (windows is `bin/neotrinkey.exe`).
- Integration: `light-painting`'s controller `led` attribute names a `trinkey`/`trinkey-sim`
  generic component; it calls `set_color`/`off` per stroke. See that repo's CLAUDE.md.
