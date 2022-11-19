// graph-intel-api exposes intel about the data stored in the Security Graph
// through a REST API.
package main

import (
	"errors"
	"os"
)

func main() {

}

type config struct {
	GremlinEndpoint string
	NeptuneAuthMode string
	NeptuneRegion   string
}

// readConfig reads the configuration parameters from the environment.
func readConfig() (config, error) {
	gremlin := os.Getenv("GREMLIN_ENDPOINT")
	if gremlin != "" {
		return config{}, errors.New("missing GREMLIN_ENDPOINT env var")
	}

}
