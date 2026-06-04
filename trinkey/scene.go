package trinkey

import (
	"context"
	"fmt"
	"sync"

	"github.com/golang/geo/r3"
	visuals "github.com/viam-labs/viam-viz-helpers-go"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/worldstatestore"
)

// SceneModel is a world_state_store visualizer that renders the 4 NeoPixels as
// a row of spheres: each sphere's SIZE encodes the pixel's intensity (off = a
// small dark dot, full brightness = a large glowing ball) and its COLOR encodes
// the pixel's RGB.
var SceneModel = resource.NewModel("viam", "neotrinkey", "trinkey-scene")

const (
	minRadiusMM = 6.0
	maxRadiusMM = 28.0
)

func init() {
	resource.RegisterService(worldstatestore.API, SceneModel,
		resource.Registration[worldstatestore.Service, *SceneConfig]{
			Constructor: newScene,
		},
	)
}

type vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// SceneConfig positions the LED row in the world.
type SceneConfig struct {
	// Origin is the world position (mm) of the first LED. Defaults to {0,300,300}.
	Origin *vec3 `json:"origin,omitempty"`
	// SpacingMM is the spacing between LEDs along +Y. Defaults to 45.
	SpacingMM float64 `json:"spacing_mm,omitempty"`
}

func (c *SceneConfig) Validate(string) ([]string, []string, error) { return nil, nil, nil }

// Sink is the in-process interface the driver uses to push LED state.
type Sink interface {
	ShowPixels(pixels [NumPixels]RGB, brightness float64)
}

// Lookup returns the in-process trinkey-scene registered under name, if any.
func Lookup(name string) (Sink, bool) {
	s, ok := visuals.Lookup(name).(Sink)
	return s, ok
}

type ledScene struct {
	resource.Named
	visuals.SceneServiceBase
	logger logging.Logger

	mu         sync.Mutex
	origin     r3.Vector
	spacing    float64
	pixels     [NumPixels]RGB
	brightness float64
}

func newScene(
	ctx context.Context, deps resource.Dependencies, conf resource.Config, logger logging.Logger,
) (worldstatestore.Service, error) {
	s := &ledScene{logger: logger, brightness: 1.0}
	s.Named = conf.ResourceName().AsNamed()
	s.SceneServiceBase.Hooks = s
	s.SceneServiceBase.Logger = logger
	s.SceneServiceBase.DefaultParentFrame = "world"
	if err := s.Reconfigure(ctx, deps, conf); err != nil {
		return nil, err
	}
	logger.Info("neotrinkey trinkey-scene visualizer ready")
	return s, nil
}

func (s *ledScene) Reconfigure(_ context.Context, _ resource.Dependencies, conf resource.Config) error {
	cfg, err := resource.NativeConfig[*SceneConfig](conf)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.origin = r3.Vector{X: 0, Y: 300, Z: 300}
	if cfg.Origin != nil {
		s.origin = r3.Vector{X: cfg.Origin.X, Y: cfg.Origin.Y, Z: cfg.Origin.Z}
	}
	s.spacing = 45
	if cfg.SpacingMM > 0 {
		s.spacing = cfg.SpacingMM
	}
	s.mu.Unlock()

	if err := s.SceneServiceBase.ReconfigureWith(nil, 0, "", "world"); err != nil {
		return err
	}
	visuals.Register(conf.ResourceName().Name, s)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.rebuildLocked()
	return nil
}

// DoCommand disambiguates SceneServiceBase's verbs from resource.Named.
func (s *ledScene) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return s.SceneServiceBase.DoCommand(ctx, cmd)
}

func (s *ledScene) Close(ctx context.Context) error {
	visuals.Unregister(s.Named.Name().Name)
	return s.SceneServiceBase.Close(ctx)
}

// ShowPixels updates the LED visuals from the driver.
func (s *ledScene) ShowPixels(pixels [NumPixels]RGB, brightness float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pixels = pixels
	s.brightness = brightness
	s.rebuildLocked()
}

func (s *ledScene) rebuildLocked() {
	vs := make([]interface{}, 0, NumPixels)
	for i, p := range s.pixels {
		intensity := float64(maxU8(p.R, p.G, p.B)) / 255.0 * s.brightness
		radius := minRadiusMM + intensity*(maxRadiusMM-minRadiusMM)
		col := visuals.Color{R: int(p.R), G: int(p.G), B: int(p.B)}
		if p.R == 0 && p.G == 0 && p.B == 0 {
			col = visuals.Color{R: 38, G: 38, B: 48} // dark dot when off
		}
		pos := s.origin.Add(r3.Vector{Y: float64(i) * s.spacing})
		vs = append(vs, &visuals.Sphere{
			Label:    fmt.Sprintf("led_%d", i),
			Pose:     visuals.PoseAt(pos.X, pos.Y, pos.Z, 0, 0, 1, 0),
			RadiusMM: radius,
			Color:    &col,
		})
	}
	if err := s.SceneServiceBase.SetScene(visuals.SetSceneOpts{ParentFrame: "world"}, vs...); err != nil {
		s.logger.Warnw("trinkey-scene SetScene failed", "err", err)
	}
}

func maxU8(a, b, c uint8) uint8 {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}
