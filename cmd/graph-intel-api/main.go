// graph-intel-api exposes intel about the data stored in the Security Graph
// through a REST API.
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/adevinta/graph-intel-api/intel"
	"github.com/adevinta/graph-intel-api/log"
	"github.com/julienschmidt/httprouter"
)

const defaultLogLevel = "info"

func main() {
	cfg, err := readConfig()
	if err != nil {
		log.Fatalf("graph-intel-api: error reading config: %v", err)
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

// intel defines the shape of the intel API exposed by the intelRESTAPI.
type intelAPI interface {
	BlastRadius(identifier string, assetType string) (intel.BlastRadiusResult, error)
}

// intelRESTAPI exposes the Security Graph intel API as an HTTP REST endpoint.
type intelRESTAPI struct {
	router *httprouter.Router
	intel  intelAPI
}

// newIntelRESTAPI creates a new intel REST API that exposed the given
// Security Graph intel API.
func newIntelRESTAPI(intel intelAPI) *intelRESTAPI {
	router := httprouter.New()
	api := &intelRESTAPI{
		intel:  intel,
		router: router,
	}
	router.GET("blast-radius/", api.BlastRadius)
	return api
}

func (i *intelRESTAPI) BlastRadius(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	assetType := ps.ByName("asset_type")
	if assetType == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	identifier := ps.ByName("asset_identifier")
	if identifier == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	
	err, b := i.intel.BlastRadius(identifier, assetType)
	if err := nil {
		
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(b)
}
