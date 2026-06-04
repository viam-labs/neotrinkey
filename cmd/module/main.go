// Command module runs the neotrinkey module: the real serial driver, the
// simulated driver, and the 4-LED world_state_store visualizer.
package main

import (
	"neotrinkey/trinkey"

	"go.viam.com/rdk/components/generic"
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/worldstatestore"
)

func main() {
	module.ModularMain(
		resource.APIModel{API: generic.API, Model: trinkey.Model},
		resource.APIModel{API: generic.API, Model: trinkey.SimModel},
		resource.APIModel{API: worldstatestore.API, Model: trinkey.SceneModel},
	)
}
