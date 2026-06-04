// Package trinkey implements Viam models for the Adafruit NeoTrinkey
// (SAMD21 USB key with 4 NeoPixels): a real serial driver, a simulated driver,
// and a world_state_store visualizer for the 4 LEDs.
//
// https://www.adafruit.com/product/4870
package trinkey

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

// NumPixels is the number of NeoPixels on the NeoTrinkey.
const NumPixels = 4

// RGB is an 8-bit-per-channel color.
type RGB struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

// writer is the hardware-facing side of the driver: simulated (in-memory) or
// real (USB serial). write receives the brightness-applied colors.
type writer interface {
	write(pixels [NumPixels]RGB) error
	close() error
}

// driver is the shared NeoTrinkey driver behind both the sim and real models.
// It is a generic component whose DoCommand drives the 4 pixels.
type driver struct {
	resource.Named
	resource.AlwaysRebuild

	logger logging.Logger
	w      writer
	scene  Sink // optional 3D LED visualizer; nil if not configured

	mu         sync.Mutex
	pixels     [NumPixels]RGB
	brightness float64
}

func newDriver(name resource.Name, logger logging.Logger, w writer, scene Sink) *driver {
	d := &driver{
		Named:      name.AsNamed(),
		logger:     logger,
		w:          w,
		scene:      scene,
		brightness: 1.0,
	}
	d.mu.Lock()
	_ = d.flushLocked() // push the initial (off) state to hw + scene
	d.mu.Unlock()
	return d
}

// DoCommand drives the pixels. Supported commands (via "command" or a bare key):
//
//	set_pixels  {"pixels": [[r,g,b], ...]}   set pixels by index (others unchanged)
//	set_all     {"color": [r,g,b]}           set all 4 pixels (alias: set_color)
//	off         {}                           all pixels off
//	brightness  {"brightness": 0..1}         global brightness multiplier
//	get_pixels  {}                           return current pixels + brightness
func (d *driver) DoCommand(_ context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	action, _ := cmd["command"].(string)
	if action == "" {
		for _, k := range []string{"set_pixels", "set_all", "set_color", "off", "brightness", "get_pixels"} {
			if _, ok := cmd[k]; ok {
				action = k
				break
			}
		}
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	switch action {
	case "set_pixels":
		ps, err := parsePixelList(cmd["pixels"])
		if err != nil {
			return nil, err
		}
		for i, c := range ps {
			if i < NumPixels {
				d.pixels[i] = c
			}
		}
		return d.applyLocked()
	case "set_all", "set_color":
		c, err := parseColor(cmd["color"])
		if err != nil {
			return nil, err
		}
		for i := range d.pixels {
			d.pixels[i] = c
		}
		return d.applyLocked()
	case "off":
		d.pixels = [NumPixels]RGB{}
		return d.applyLocked()
	case "brightness":
		b, ok := toFloat(cmd["brightness"])
		if !ok {
			return nil, fmt.Errorf("brightness must be a number in [0,1]")
		}
		d.brightness = clamp01(b)
		return d.applyLocked()
	case "get_pixels":
		return d.stateLocked(), nil
	default:
		return nil, fmt.Errorf(
			"unknown command %q; supported: set_pixels, set_all/set_color, off, brightness, get_pixels", action)
	}
}

func (d *driver) Close(context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pixels = [NumPixels]RGB{} // turn the light off on the way out
	_ = d.flushLocked()
	return d.w.close()
}

// applyLocked flushes to hw + scene and returns the new state. Caller holds mu.
func (d *driver) applyLocked() (map[string]interface{}, error) {
	if err := d.flushLocked(); err != nil {
		return nil, err
	}
	return d.stateLocked(), nil
}

// flushLocked pushes the brightness-applied colors to the device and the raw
// pixels + brightness to the visualizer. Caller holds mu.
func (d *driver) flushLocked() error {
	if d.scene != nil {
		d.scene.ShowPixels(d.pixels, d.brightness)
	}
	return d.w.write(d.effectiveLocked())
}

func (d *driver) effectiveLocked() [NumPixels]RGB {
	var out [NumPixels]RGB
	for i, p := range d.pixels {
		out[i] = RGB{
			R: scale(p.R, d.brightness),
			G: scale(p.G, d.brightness),
			B: scale(p.B, d.brightness),
		}
	}
	return out
}

func (d *driver) stateLocked() map[string]interface{} {
	px := make([]map[string]interface{}, NumPixels)
	for i, p := range d.pixels {
		px[i] = map[string]interface{}{"r": p.R, "g": p.G, "b": p.B}
	}
	return map[string]interface{}{"pixels": px, "brightness": d.brightness}
}

// ---- parsing helpers --------------------------------------------------------

func parseColor(v interface{}) (RGB, error) {
	switch t := v.(type) {
	case []interface{}:
		if len(t) < 3 {
			return RGB{}, fmt.Errorf("color array must be [r,g,b]")
		}
		return RGB{R: u8(t[0]), G: u8(t[1]), B: u8(t[2])}, nil
	case map[string]interface{}:
		return RGB{R: u8(t["r"]), G: u8(t["g"]), B: u8(t["b"])}, nil
	default:
		return RGB{}, fmt.Errorf("color must be [r,g,b] or {r,g,b}, got %T", v)
	}
}

func parsePixelList(v interface{}) ([]RGB, error) {
	arr, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("pixels must be an array of colors")
	}
	out := make([]RGB, 0, len(arr))
	for _, e := range arr {
		c, err := parseColor(e)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func u8(v interface{}) uint8 {
	f, _ := toFloat(v)
	if f < 0 {
		f = 0
	}
	if f > 255 {
		f = 255
	}
	return uint8(f)
}

func scale(v uint8, b float64) uint8 {
	x := float64(v) * b
	if x < 0 {
		x = 0
	}
	if x > 255 {
		x = 255
	}
	return uint8(x)
}

func clamp01(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}
