package print

import (
	"os"

	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/ojson"
	"gopkg.in/yaml.v3"
)

// YAML implements a YAML formatter.
//
// Its zero-value is ready for use.
type YAML struct{}

// Encode allows external callers to use the same encoder
func (YAML) Encode(obj interface{}) {
	yaml.NewEncoder(os.Stdout).Encode(obj)
}

// APIKeys implementation.
func (YAML) apiKeys(apiKeys []api.APIKey) {
	yaml.NewEncoder(os.Stdout).Encode(apiKeys)
}

// Tasks implementation.
func (YAML) tasks(tasks []api.Task) {
	yaml.NewEncoder(os.Stdout).Encode(printTasks(tasks))
}

// Task implementation.
func (YAML) task(task api.Task) {
	yaml.NewEncoder(os.Stdout).Encode(printTask(task))
}

// Runs implementation.
func (YAML) runs(runs []api.Run) {
	yaml.NewEncoder(os.Stdout).Encode(runs)
}

// Run implementation.
func (YAML) run(run api.Run) {
	yaml.NewEncoder(os.Stdout).Encode(run)
}

// Outputs implementation.
func (YAML) outputs(outputs api.Outputs) {
	// TODO: update ojson to handle yaml properly
	yaml.NewEncoder(os.Stdout).Encode(ojson.Value(outputs).V)
}

// Config implementation.
func (YAML) config(config api.Config) {
	yaml.NewEncoder(os.Stdout).Encode(config)
}
