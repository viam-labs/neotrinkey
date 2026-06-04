package trinkey

import (
	"context"
	"testing"

	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

// recordWriter captures the last colors written, for assertions.
type recordWriter struct {
	last   [NumPixels]RGB
	writes int
}

func (w *recordWriter) write(p [NumPixels]RGB) error {
	w.last = p
	w.writes++
	return nil
}
func (w *recordWriter) close() error { return nil }

func newTestDriver(t *testing.T) (*driver, *recordWriter) {
	t.Helper()
	w := &recordWriter{}
	d := newDriver(resource.NewName(resource.APINamespaceRDK.WithComponentType("generic"), "t"),
		logging.NewTestLogger(t), w, nil)
	return d, w
}

func TestSetAllAndOff(t *testing.T) {
	d, w := newTestDriver(t)
	ctx := context.Background()

	if _, err := d.DoCommand(ctx, map[string]interface{}{
		"command": "set_all", "color": []interface{}{255.0, 0.0, 128.0},
	}); err != nil {
		t.Fatal(err)
	}
	for i, p := range w.last {
		if p != (RGB{R: 255, G: 0, B: 128}) {
			t.Errorf("pixel %d = %v, want {255,0,128}", i, p)
		}
	}

	if _, err := d.DoCommand(ctx, map[string]interface{}{"command": "off"}); err != nil {
		t.Fatal(err)
	}
	if w.last != ([NumPixels]RGB{}) {
		t.Errorf("after off = %v, want all zero", w.last)
	}
}

func TestSetPixelsByIndex(t *testing.T) {
	d, w := newTestDriver(t)
	_, err := d.DoCommand(context.Background(), map[string]interface{}{
		"command": "set_pixels",
		"pixels": []interface{}{
			[]interface{}{255.0, 0.0, 0.0},
			map[string]interface{}{"r": 0.0, "g": 255.0, "b": 0.0},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if w.last[0] != (RGB{R: 255}) || w.last[1] != (RGB{G: 255}) {
		t.Errorf("pixels = %v, want [red, green, off, off]", w.last)
	}
	if w.last[2] != (RGB{}) || w.last[3] != (RGB{}) {
		t.Errorf("unspecified pixels should stay off, got %v", w.last)
	}
}

func TestBrightnessScalesOutput(t *testing.T) {
	d, w := newTestDriver(t)
	ctx := context.Background()
	if _, err := d.DoCommand(ctx, map[string]interface{}{"command": "set_all", "color": []interface{}{200.0, 100.0, 50.0}}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.DoCommand(ctx, map[string]interface{}{"command": "brightness", "brightness": 0.5}); err != nil {
		t.Fatal(err)
	}
	// Effective output is halved, but reported raw state is unchanged.
	if got := w.last[0]; got != (RGB{R: 100, G: 50, B: 25}) {
		t.Errorf("dimmed output = %v, want {100,50,25}", got)
	}
	resp, _ := d.DoCommand(ctx, map[string]interface{}{"command": "get_pixels"})
	if resp["brightness"] != 0.5 {
		t.Errorf("brightness = %v, want 0.5", resp["brightness"])
	}
}

func TestUnknownCommand(t *testing.T) {
	d, _ := newTestDriver(t)
	if _, err := d.DoCommand(context.Background(), map[string]interface{}{"command": "explode"}); err == nil {
		t.Error("expected error for unknown command")
	}
}
