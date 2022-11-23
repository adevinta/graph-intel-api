// Package rest exposed the intel API using HTTP REST.
package rest

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/adevinta/graph-intel-api/api"
	"github.com/adevinta/graph-intel-api/log"
)

type restError struct {
	status int    `json:"-"`
	Msg    string `json:"msg"`
}

func (r restError) Error() string {
	return r.Msg
}

func (r restError) write(w http.ResponseWriter, req *http.Request) {
	log.Error.Printf("graph-intel-api error - error serving request to %s: %v", req.RequestURI, r)
	w.WriteHeader(r.status)
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(r)
	if err != nil {
		log.Error.Printf("graph-intel-api - error generating response for request to %s: %v", req.RequestURI, err)
	}
}

var (
	errBlastRadiusNoAssetType = restError{
		status: http.StatusBadRequest,
		Msg:    "the asset_type parameter is mandatory",
	}

	errBlastRadiusNoIdentifier = restError{
		status: http.StatusBadRequest,
		Msg:    "the asset_identifier parameter is mandatory",
	}
)

// IntelAPI defines the shape of the intel API exposed by a [Server].
type IntelAPI interface {
	BlastRadius(identifier string, assetType string) (api.BlastRadiusResult, error)
}

// Server exposes the Security Graph intel API as an HTTP Server endpoint.
type Server struct {
	router *httprouter.Router
	intel  IntelAPI
}

// NewServer creates a new intel REST API that exposes the given
// Security Graph intel API.
func NewServer(intel IntelAPI) *Server {
	router := httprouter.New()
	api := &Server{
		intel:  intel,
		router: router,
	}
	router.GET("blast-radius/", api.BlastRadius)
	return api
}

// ServeHTTP server the routes exposed by the REST API.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) BlastRadius(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	assetType := ps.ByName("asset_type")
	if assetType == "" {
		errBlastRadiusNoAssetType.write(w, r)
		return
	}
	identifier := ps.ByName("asset_identifier")
	if identifier == "" {
		errBlastRadiusNoAssetType.write(w, r)
		return
	}

	b, err := s.intel.BlastRadius(identifier, assetType)
	// TODO: check if the returned error is an `assets not found error`` or an
	// invalid a `not enough information to calculate the score` error
	if err != nil {
		err := restError{
			status: http.StatusInternalServerError,
			Msg:    err.Error(),
		}
		err.write(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(b)
	if err != nil {
		err := restError{
			status: http.StatusInternalServerError,
			Msg:    err.Error(),
		}
		err.write(w, r)
		return
	}
}
