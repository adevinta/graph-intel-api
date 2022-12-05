// Package rest exposed the intel API using HTTP REST.
package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/adevinta/graph-intel-api/intel"
	"github.com/adevinta/graph-intel-api/log"
)

// restError represents a REST error. It is serialized an returned to the user.
type restError struct {
	status int `json:"-"`

	// Msg is the error message provided in the body.
	Msg string `json:"msg"`
}

func (r restError) Error() string {
	return r.Msg
}

// write writes an error response.
func (r restError) write(w http.ResponseWriter, req *http.Request) {
	log.Error.Printf("graph-intel-api: rest: error serving request to %s: %v", req.RequestURI, r)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(r.status)

	if err := json.NewEncoder(w).Encode(r); err != nil {
		log.Error.Printf("graph-intel-api: rest: error generating response for request to %s: %v", req.RequestURI, err)
	}
}

var (
	// errMissingParameter is an error returned by the REST API when a
	// mandatory parameter is missing.
	errMissingParameter = restError{
		status: http.StatusBadRequest,
		Msg:    "missing parameter",
	}

	// errNotFound is an error returned by the REST API when an entity is
	// not found.
	errNotFound = restError{
		status: http.StatusNotFound,
		Msg:    "not found",
	}

	// errInternalServerError is an error returned by the REST API when the
	// server fails handling a request.
	errInternalServerError = restError{
		status: http.StatusInternalServerError,
		Msg:    "internal server error",
	}
)

// IntelAPI includes the method set of [intel.API]. We do not expect to have
// multiple implementations, but depending on an interface makes easier to test
// this package.
type IntelAPI interface {
	// BlastRadius returns the blast radius of a given asset.
	BlastRadius(typ, identifier string) (intel.BlastRadiusResult, error)
}

// API exposes the Security Graph intel API as an HTTP REST endpoint.
type API struct {
	router   *httprouter.Router
	intelAPI IntelAPI
}

// NewAPI creates a new intel REST API that exposes the given Security
// Graph intel API.
func NewAPI(intelAPI IntelAPI) API {
	router := httprouter.New()
	api := API{
		intelAPI: intelAPI,
		router:   router,
	}
	router.GET("/v1/blast-radius", api.BlastRadius)
	return api
}

// ServeHTTP serves the routes exposed by the REST API.
func (api API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.router.ServeHTTP(w, r)
}

// BlastRadius handles the endpoint that returns the blast radius given a
// specific asset.
func (api API) BlastRadius(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	params := r.URL.Query()
	typ := params.Get("asset_type")
	if typ == "" {
		errMissingParameter.write(w, r)
		return
	}
	identifier := params.Get("asset_identifier")
	if identifier == "" {
		errMissingParameter.write(w, r)
		return
	}

	br, err := api.intelAPI.BlastRadius(typ, identifier)
	if err != nil {
		if errors.Is(err, intel.ErrNotFound) {
			errNotFound.write(w, r)
			return
		}
		log.Error.Printf("graph-intel-api: rest: error calculating Blast Radius: %v", err)
		errInternalServerError.write(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(br); err != nil {
		errInternalServerError.write(w, r)
		return
	}
}
