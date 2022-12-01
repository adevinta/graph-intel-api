// graph-intel-api exposes intel about the data stored in the Security Graph
// through a REST API.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/adevinta/graph-intel-api/gremlin"
	"github.com/adevinta/graph-intel-api/intel"
	"github.com/adevinta/graph-intel-api/log"
	"github.com/adevinta/graph-intel-api/rest"
)

const (
	defaultLogLevel                  = "info"
	defaultListenAddr                = ":8000"
	defaultGremlinAuthMode           = "none"
	defaultAWSRegion                 = "eu-west-1"
	defaultGremlinRetryLimit         = 5
	defaultGremlinRetryDuration      = 5 * time.Second
	defaultIntelResolveTimeoutMs     = 60000
	defaultIntelBlastRadiusTimeoutMs = 60000
)

func main() {
	cfg, err := readConfig()
	if err != nil {
		log.Fatalf("graph-intel-api: error reading config: %v", err)
	}

	if err := runAndServe(context.Background(), cfg); err != nil {
		log.Fatalf("graph-intel-api: error running server %v", err)
	}
}

// runAndServe serves the actual API.
func runAndServe(ctx context.Context, cfg config) error {
	if err := log.SetLevel(cfg.LogLevel); err != nil {
		return fmt.Errorf("error setting log level: %w", err)
	}

	intelAPI, err := intel.NewAPI(cfg.IntelConfig)
	if err != nil {
		return fmt.Errorf("error creating intel API: %w", err)
	}

	restAPI := rest.NewAPI(intelAPI)
	mux := http.NewServeMux()
	mux.Handle("/", restAPI)

	log.Info.Printf("graph-intel-api: listening on address %s", cfg.ListenAddr)
	err = http.ListenAndServe(cfg.ListenAddr, mux)
	if errors.Is(err, http.ErrServerClosed) {
		log.Info.Printf("graph-intel-api: server closed")
		return nil
	}
	return err
}

// config defines the config parameters used by graph-intel-api.
type config struct {
	LogLevel    string
	ListenAddr  string
	IntelConfig intel.Config
}

// readConfig reads the configuration parameters from the environment.
func readConfig() (cfg config, err error) {
	// Required configuration.

	gremlinEndpoint := os.Getenv("GREMLIN_ENDPOINT")
	if gremlinEndpoint == "" {
		return config{}, errors.New("missing GREMLIN_ENDPOINT env var")
	}

	// Optional configuration.

	logLevel := defaultLogLevel
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		logLevel = level
	}

	listenAddr := defaultListenAddr
	if addr := os.Getenv("LISTEN_ADDR"); addr != "" {
		listenAddr = addr
	}

	gremlinAuthMode := defaultGremlinAuthMode
	if mode := os.Getenv("GREMLIN_AUTH_MODE"); mode != "" {
		gremlinAuthMode = mode
	}

	awsRegion := defaultAWSRegion
	if region := os.Getenv("AWS_REGION"); region != "" {
		awsRegion = region
	}

	gremlinRetryLimit := defaultGremlinRetryLimit
	if limit := os.Getenv("GREMLIN_RETRY_LIMIT"); limit != "" {
		gremlinRetryLimit, err = strconv.Atoi(limit)
		if err != nil {
			return config{}, fmt.Errorf("invalid GREMLIN_RETRY_LIMIT value")
		}
	}

	gremlinRetryDuration := defaultGremlinRetryDuration
	if duration := os.Getenv("GREMLIN_RETRY_DURATION"); duration != "" {
		gremlinRetryDuration, err = time.ParseDuration(duration)
		if err != nil {
			return config{}, fmt.Errorf("invalid GREMLIN_RETRY_DURATION value")
		}
	}

	intelResolveTimeoutMs := defaultIntelResolveTimeoutMs
	if timeout := os.Getenv("INTEL_RESOLVE_TIMEOUT_MS"); timeout != "" {
		intelResolveTimeoutMs, err = strconv.Atoi(timeout)
		if err != nil {
			return config{}, fmt.Errorf("invalid INTEL_RESOLVE_TIME_MS value")
		}
	}

	intelBlastRadiusTimeoutMs := defaultIntelBlastRadiusTimeoutMs
	if timeout := os.Getenv("INTEL_BLAST_RADIUS_TIMEOUT_MS"); timeout != "" {
		intelBlastRadiusTimeoutMs, err = strconv.Atoi(timeout)
		if err != nil {
			return config{}, fmt.Errorf("invalid INTEL_BLAST_RADIUS_TIMEOUT_MS value")
		}
	}

	cfg = config{
		LogLevel:   logLevel,
		ListenAddr: listenAddr,
		IntelConfig: intel.Config{
			GremlinConfig: gremlin.Config{
				Endpoint:      gremlinEndpoint,
				AuthMode:      gremlinAuthMode,
				AWSRegion:     awsRegion,
				RetryLimit:    gremlinRetryLimit,
				RetryDuration: gremlinRetryDuration,
			},
			ResolveTimeoutMs:     intelResolveTimeoutMs,
			BlastRadiusTimeoutMs: intelBlastRadiusTimeoutMs,
		},
	}
	return cfg, nil
}
