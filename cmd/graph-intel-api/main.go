// graph-intel-api exposes intel about the data stored in the Security Graph
// through a REST API.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/adevinta/graph-intel-api/api"
	"github.com/adevinta/graph-intel-api/log"
	"github.com/adevinta/graph-intel-api/rest"
)

const defaultLogLevel = "info"

func main() {
	cfg, err := readConfig()
	if err != nil {
		log.Fatalf("graph-intel-api: error reading config: %v", err)
	}

	err = runAndServe(context.Background(), cfg)
	if err != nil {
		log.Error.Fatalf("graph-intel-api - error running server %v", err)
	}
}

func runAndServe(ctx context.Context, cfg config) error {
	err := log.SetLevel(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("error setting log level: %w", err)
	}
	api := api.NewAPI(cfg.Config)
	restAPI := rest.NewServer(api)
	mux := http.NewServeMux()
	mux.Handle("/", restAPI)
	log.Info.Printf("graph-intel-api - listening on address %s", cfg.ListenAddr)
	err = http.ListenAndServe(cfg.ListenAddr, mux)
	if err == http.ErrServerClosed {
		log.Info.Printf("graph-intel-api - server closed")
		return nil
	}
	return err
}

// config defines the config parameters used by graph-intel-api.
type config struct {
	LogLevel   string
	ListenAddr string
	api.Config
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
	addr := os.Getenv("LISTER_ADDR")
	if addr == "" {
		addr = ":8085"
	}
	logLevel := defaultLogLevel
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		logLevel = level
	}
	cfg := config{
		LogLevel:   logLevel,
		ListenAddr: addr,
		Config: api.Config{
			GremlinEndpoint: gremlin,
			NeptuneAuthMode: neptuneAuth,
			NeptuneRegion:   neptuneRegion,
		},
	}
	return cfg, nil
}
