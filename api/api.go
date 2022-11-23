// Package intel provides the API for getting all the intel information
// exposed by the Security Graph.
package api

import "errors"

var (
	ErrAssetNotFound     = errors.New("asset not found")
	ErrNoBlastRadiusInfo = errors.New("not enough information")
)

// Config contains the configuration params needed by the intel functions.
type Config struct {
	GremlinEndpoint string
	NeptuneAuthMode string
	NeptuneRegion   string
}

type BlastRadiusResult struct {
	// Score contains the blast radius score for a given asset.
	Score float32
	// Metadata contains information about how a blast radius was calculated.
	Metadata string
}

// BlastRadius returns the blast radius of a given asset. It returns
func BlastRadius(cfg Config, identifier string, assetType string) (BlastRadiusResult, error) {
	return BlastRadiusResult{}, errors.New("not implemented")
}
