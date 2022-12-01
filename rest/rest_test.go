package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adevinta/graph-intel-api/intel"

	"github.com/google/go-cmp/cmp"
)

func TestAPIBlastRadius(t *testing.T) {
	tests := []struct {
		name       string
		mock       blastRadiusMock
		params     blastRadiusParams
		wantStatus int
		wantResp   blastRadiusResp
	}{
		{
			name: "ok",
			mock: blastRadiusMock{
				typ:        "typ1",
				identifier: "identifier1",
				score:      123.123,
			},
			params: blastRadiusParams{
				typ:        "typ1",
				identifier: "identifier1",
			},
			wantStatus: http.StatusOK,
			wantResp: blastRadiusResp{
				Score:    123.123,
				Metadata: "mock",
			},
		},
		{
			name: "not found",
			mock: blastRadiusMock{
				typ:        "typ1",
				identifier: "identifier1",
				score:      123.123,
			},
			params: blastRadiusParams{
				typ:        "unknown_typ",
				identifier: "unknown_identifier",
			},
			wantStatus: http.StatusNotFound,
			wantResp:   blastRadiusResp{},
		},
		{
			name: "internal server error",
			mock: blastRadiusMock{
				forceError: true,
			},
			params: blastRadiusParams{
				typ:        "typ1",
				identifier: "identifier1",
			},
			wantStatus: http.StatusInternalServerError,
			wantResp:   blastRadiusResp{},
		},
		{
			name: "missing parameter asset_identifier",
			mock: blastRadiusMock{
				typ:        "typ1",
				identifier: "identifier1",
				score:      123.123,
			},
			params: blastRadiusParams{
				typ: "typ1",
			},
			wantStatus: http.StatusBadRequest,
			wantResp:   blastRadiusResp{},
		},
		{
			name: "missing parameter asset_type",
			mock: blastRadiusMock{
				typ:        "typ1",
				identifier: "identifier1",
				score:      123.123,
			},
			params: blastRadiusParams{
				identifier: "identifier1",
			},
			wantStatus: http.StatusBadRequest,
			wantResp:   blastRadiusResp{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restAPI := NewAPI(tt.mock)
			ts := httptest.NewServer(restAPI)
			defer ts.Close()

			url := fmt.Sprintf("%s/v1/blast-radius?asset_type=%v&asset_identifier=%v", ts.URL, tt.params.typ, tt.params.identifier)
			res, err := http.Get(url)
			if err != nil {
				t.Fatalf("request error: %v", err)
			}

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("unexpected status: got=%v want=%v", res.StatusCode, tt.wantStatus)
			}

			if tt.wantStatus != http.StatusOK {
				return
			}

			var got blastRadiusResp
			if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
				t.Fatalf("malformed body: %v", err)
			}
			res.Body.Close()

			if diff := cmp.Diff(tt.wantResp, got); diff != "" {
				t.Errorf("messages mismatch (-want +got):\n%v", diff)
			}
		})
	}
}

type blastRadiusParams struct {
	typ        string
	identifier string
}

type blastRadiusResp struct {
	Score    float64 `json:"score"`
	Metadata string  `json:"metadata"`
}

type blastRadiusMock struct {
	typ        string
	identifier string
	score      float64
	forceError bool
}

func (mock blastRadiusMock) BlastRadius(typ, identifier string) (intel.BlastRadiusResult, error) {
	if mock.forceError {
		return intel.BlastRadiusResult{}, errors.New("forced error")
	}

	if typ == mock.typ && identifier == mock.identifier {
		result := intel.BlastRadiusResult{
			Score:    mock.score,
			Metadata: "mock",
		}
		return result, nil
	}

	return intel.BlastRadiusResult{}, intel.ErrNotFound
}
