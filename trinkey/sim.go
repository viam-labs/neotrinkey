package trinkey

import (
	"context"

	"go.viam.com/rdk/components/generic"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/worldstatestore"
)

// SimModel is the simulated NeoTrinkey (no hardware).
var SimModel = resource.NewModel("viam", "neotrinkey", "trinkey-sim")

func init() {
	resource.RegisterComponent(generic.API, SimModel,
		resource.Registration[resource.Resource, *SimConfig]{
			Constructor: newSim,
		},
	)
}

// SimConfig configures the simulated driver.
type SimConfig struct {
	// Scene optionally names a trinkey-scene visualizer to draw the LEDs into.
	Scene string `json:"scene,omitempty"`
}

func (c *SimConfig) Validate(path string) ([]string, []string, error) {
	var deps []string
	if c.Scene != "" {
		deps = append(deps, worldstatestore.Named(c.Scene).String())
	}
	return deps, nil, nil
}

// simWriter just logs the pixel state — no hardware.
type simWriter struct {
	logger logging.Logger
	name   string
}

func (w simWriter) write(pixels [NumPixels]RGB) error {
	w.logger.Debugf("[trinkey-sim %s] %v", w.name, pixels)
	return nil
}

func (w simWriter) close() error { return nil }

func newSim(
	_ context.Context, _ resource.Dependencies, conf resource.Config, logger logging.Logger,
) (resource.Resource, error) {
	cfg, err := resource.NativeConfig[*SimConfig](conf)
	if err != nil {
		return nil, err
	}
	sink := lookupScene(cfg.Scene, logger)
	d := newDriver(conf.ResourceName(), logger,
		simWriter{logger: logger, name: conf.ResourceName().Name}, sink)
	logger.Info("neotrinkey trinkey-sim ready (4 NeoPixels, simulated)")
	return d, nil
}

// lookupScene resolves an optional in-process LED visualizer by name.
func lookupScene(name string, logger logging.Logger) Sink {
	if name == "" {
		return nil
	}
	if s, ok := Lookup(name); ok {
		return s
	}
	logger.Warnf("scene %q not found in-process; LED visualization disabled", name)
	return nil
}
