# neotrinkey

Viam module for the [Adafruit NeoTrinkey — SAMD21 USB Key with 4 NeoPixels](https://www.adafruit.com/product/4870).

Three models:

| Model | API | What |
|-------|-----|------|
| `viam:neotrinkey:trinkey` | `rdk:component:generic` | **Real driver** — drives the 4 NeoPixels over the NeoTrinkey's USB serial port |
| `viam:neotrinkey:trinkey-sim` | `rdk:component:generic` | **Simulated driver** — same API, no hardware |
| `viam:neotrinkey:trinkey-scene` | `rdk:service:world_state_store` | **3D visualizer** — shows the 4 LEDs as spheres (size = intensity, color = RGB) |

Built as the LED for [light-painting](https://github.com/viam-labs/light-painting): the arm
paints with the NeoPixel color, and the visualizer shows the LED state in the 3D scene.

## DoCommand API (both drivers)

| Command | Payload | Effect |
|---------|---------|--------|
| `set_pixels` | `{"pixels": [[r,g,b], ...]}` | set pixels by index (0–3); unspecified stay |
| `set_all` / `set_color` | `{"color": [r,g,b]}` (or `{r,g,b}`) | set all 4 pixels |
| `off` | – | all pixels off |
| `brightness` | `{"brightness": 0..1}` | global brightness multiplier |
| `get_pixels` | – | returns `{pixels, brightness}` |

The `trinkey-scene` (if configured via the driver's `scene` attribute) updates live: each
LED is a sphere whose radius scales with `max(r,g,b)·brightness` and whose color is the
pixel RGB (a small dark dot when off). Both models ship in one binary, so the driver pushes
to the visualizer in-process (no gRPC).

## Configuration

```json
{
  "components": [
    {
      "name": "led",
      "api": "rdk:component:generic",
      "model": "viam:neotrinkey:trinkey",
      "attributes": { "serial_path": "/dev/ttyACM1", "baud": 115200, "scene": "led-scene" }
    }
  ],
  "services": [
    { "name": "led-scene", "api": "rdk:service:world_state_store", "model": "viam:neotrinkey:trinkey-scene",
      "attributes": { "origin": {"x": 0, "y": 300, "z": 300}, "spacing_mm": 45 } }
  ]
}
```

Swap `model` to `viam:neotrinkey:trinkey-sim` (and drop `serial_path`) for the simulator.

## Real hardware setup (firmware)

The real driver talks to the NeoTrinkey over its **USB CDC data** serial port. Flash the
device with CircuitPython + the `neopixel` library, then copy `firmware/boot.py` and
`firmware/code.py` onto the `CIRCUITPY` drive and reset. The firmware enables the data
serial channel and parses `r,g,b,r,g,b,r,g,b,r,g,b\n` lines from the host.

Find the data port (the second `ttyACM`/`cu.usbmodem` the NeoTrinkey exposes) and set it as
`serial_path`. USB CDC ignores baud, but a value is still required (default 115200).

## Build

```bash
make            # test + build bin/neotrinkey + module.tar.gz (current platform)
make test       # unit tests (pixel commands, brightness)
make build-all  # cross-compile all platforms -> dist/neotrinkey-<os>-<arch>.tar.gz
```

Supported platforms: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`,
`windows/amd64` (pure Go — `go.bug.st/serial`, no cgo).
