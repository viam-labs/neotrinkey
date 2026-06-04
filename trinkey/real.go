package trinkey

import (
	"context"
	"fmt"
	"sync"

	"go.bug.st/serial"
	"go.viam.com/rdk/components/generic"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/worldstatestore"
)

// Model is the real NeoTrinkey, driven over its USB serial (CDC) port. It
// expects the device to run the CircuitPython firmware in firmware/ (boot.py
// enables usb_cdc.data; code.py parses "r,g,b,..,b\n" lines and sets the pixels).
var Model = resource.NewModel("viam", "neotrinkey", "trinkey")

const defaultBaud = 115200

func init() {
	resource.RegisterComponent(generic.API, Model,
		resource.Registration[resource.Resource, *Config]{
			Constructor: newReal,
		},
	)
}

// Config configures the real serial driver.
type Config struct {
	// SerialPath is the device path of the NeoTrinkey's data serial port
	// (e.g. /dev/ttyACM1 on Linux, /dev/cu.usbmodemXXXX on macOS, COM5 on Windows).
	SerialPath string `json:"serial_path"`
	// Baud is the serial baud rate. USB CDC ignores it, but it must be set;
	// defaults to 115200.
	Baud int `json:"baud,omitempty"`
	// Scene optionally names a trinkey-scene visualizer to draw the LEDs into.
	Scene string `json:"scene,omitempty"`
}

func (c *Config) Validate(path string) ([]string, []string, error) {
	if c.SerialPath == "" {
		return nil, nil, resource.NewConfigValidationFieldRequiredError(path, "serial_path")
	}
	var deps []string
	if c.Scene != "" {
		deps = append(deps, worldstatestore.Named(c.Scene).String())
	}
	return deps, nil, nil
}

// serialWriter writes "r,g,b,r,g,b,r,g,b,r,g,b\n" lines to the device.
type serialWriter struct {
	mu   sync.Mutex
	port serial.Port
}

func (w *serialWriter) write(p [NumPixels]RGB) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	line := fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d\n",
		p[0].R, p[0].G, p[0].B,
		p[1].R, p[1].G, p[1].B,
		p[2].R, p[2].G, p[2].B,
		p[3].R, p[3].G, p[3].B)
	_, err := w.port.Write([]byte(line))
	return err
}

func (w *serialWriter) close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.port == nil {
		return nil
	}
	return w.port.Close()
}

func newReal(
	_ context.Context, _ resource.Dependencies, conf resource.Config, logger logging.Logger,
) (resource.Resource, error) {
	cfg, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return nil, err
	}
	baud := cfg.Baud
	if baud == 0 {
		baud = defaultBaud
	}
	port, err := serial.Open(cfg.SerialPath, &serial.Mode{BaudRate: baud})
	if err != nil {
		return nil, fmt.Errorf("opening NeoTrinkey serial port %q: %w", cfg.SerialPath, err)
	}
	sink := lookupScene(cfg.Scene, logger)
	d := newDriver(conf.ResourceName(), logger, &serialWriter{port: port}, sink)
	logger.Infof("neotrinkey trinkey ready on %s @ %d baud", cfg.SerialPath, baud)
	return d, nil
}
