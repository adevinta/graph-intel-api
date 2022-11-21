// graph-intel-api exposes intel about the data stored in the Security Graph
// through a REST API.
package main

import (
	"errors"
	"os"

	"github.com/adevinta/graph-intel-api/log"
)

const defaultLogLevel = "info"

func main() {
	cfg, err := readConfig()
	if err != nil {
		log.Fatalf("graph-intel-api: error reading config: %v", err)
	}
}

type config struct {
	LogLevel                  string
	GremlinEndpoint           string
	NeptuneAuthMode           string
	NeptuneRegion             string
	AssetInventoryAPIEndpoint string
}

// readConfig reads the configuration parameters from the environment.
func readConfig() (config, error) {
	inventoryEndpoint := os.Getenv("INVENTORY_ENDPOINT")
	if inventoryEndpoint == "" {
		return config{}, errors.New("missing INVENTORY_ENDPOINT env var")
	}
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
		LogLevel:                  logLevel,
		GremlinEndpoint:           gremlin,
		NeptuneAuthMode:           neptuneAuth,
		AssetInventoryAPIEndpoint: inventoryEndpoint,
	}
	return cfg, nil
}
