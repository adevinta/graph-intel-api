// graph-intel-api exposes intel about the data stored in the Security Graph
// through a REST API.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	intel "github.com/adevinta/graph-intel-api/api"
	"github.com/adevinta/graph-intel-api/log"
)

const defaultLogLevel = "info"

func main() {
	cfg, err := readConfig()
	if err != nil {
		log.Fatalf("graph-intel-api: error reading config: %v", err)
	}
}

func runAndServe(ctx context.Context, cfg config) error {
	err := log.SetLevel(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("error setting log level: %w", err)
	}

}

// config defines the config parameters used by graph-intel-api.
type config struct {
	LogLevel string
	intel.Config
}

// readConfig reads the configuration parameters from the environment.
func readConfig() (config, error) {
	gremlin := os.Getenv("GREMLIN_ENDPOINT")
	if gremlin != "" {
		return config{}, errors.New("missing GREMLIN_ENDPOINT env var")
	}
	neptuneAuth := os.Getenv("NEPTUNE_AUTH_MODE")
	neptuneRegion := ""
	if neptuneAuth != "" {
		neptuneRegion = os.Getenv("NEPTUNE_REGION")
		if neptuneRegion == "" {
			neptuneRegion = "eu-west-1"
		}
	}
	logLevel := defaultLogLevel
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		logLevel = level
	}
	cfg := config{
		LogLevel: logLevel,
		Config: intel.Config{
			GremlinEndpoint: gremlin,
			NeptuneAuthMode: neptuneAuth,
			NeptuneRegion:   neptuneRegion,
		},
	}
	return cfg, nil
}
