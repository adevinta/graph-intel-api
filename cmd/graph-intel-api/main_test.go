package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adevinta/graph-intel-api/gremlin"
	"github.com/adevinta/graph-intel-api/intel"

	gremlingo "github.com/apache/tinkerpop/gremlin-go/v3/driver"
	"github.com/google/go-cmp/cmp"
)

const gremlinEndpoint = "ws://127.0.0.1:8182/gremlin"

type blastRadiusResp struct {
	Score    float64 `json:"score"`
	Metadata string  `json:"metadata"`
}

var wantBlastRadiusResp = blastRadiusResp{
	Score:    0.3106893106893107,
	Metadata: "net",
}

func setupGraph() error {
	gremlinConfig := gremlin.Config{
		Endpoint: gremlinEndpoint,
		AuthMode: "plain",
	}
	conn, err := gremlin.NewConnection(gremlinConfig)
	if err != nil {
		return fmt.Errorf("error creating Gremlin connection: %w", err)
	}

	_, err = conn.Query(func(g *gremlingo.GraphTraversalSource) ([]*gremlingo.Result, error) {
		<-g.V().Drop().Iterate()
		<-g.
			AddV("Universe").Property(gremlingo.T.Id, "u0").Property("namespace", "altimeter").Property("version", 1).As("u0").
			AddV("altimeter_snapshot").Property(gremlingo.T.Id, "s0").Property("timestamp", 0).As("s0").
			AddV("ec2:network-interface").Property(gremlingo.T.Id, "ni0").Property("public_ip", "1.2.3.4").Property("public_dns_name", "example.com").Property("status", "in-use").As("ni0").
			AddV("ec2:security-group").Property(gremlingo.T.Id, "sg0").As("sg0").
			AddV("egress_rule").Property(gremlingo.T.Id, "er0").As("er0").
			AddV("ip_range").Property(gremlingo.T.Id, "r0").As("r0").
			AddV("user_id_group_pairs").Property(gremlingo.T.Id, "uigp0").As("uigp0").
			AddV("ingress_rule").Property(gremlingo.T.Id, "ir0").As("ir0").
			AddV("ec2:security-group").Property(gremlingo.T.Id, "sg1").As("sg1").
			AddV("ec2:instance").Property(gremlingo.T.Id, "i0").As("i0").
			AddV("egress_rule").Property(gremlingo.T.Id, "er1").As("er1").
			AddV("ip_range").Property(gremlingo.T.Id, "r1").As("r1").
			AddE("universe_of").From("u0").To("s0").
			AddE("includes").From("s0").To("ni0").
			AddE("includes").From("s0").To("sg0").
			AddE("includes").From("s0").To("er0").
			AddE("includes").From("s0").To("r0").
			AddE("includes").From("s0").To("uigp0").
			AddE("includes").From("s0").To("ir0").
			AddE("includes").From("s0").To("sg1").
			AddE("includes").From("s0").To("i0").
			AddE("includes").From("s0").To("er1").
			AddE("includes").From("s0").To("r1").
			AddE("resource_link").From("ni0").To("sg0").
			AddE("egress_rule").From("sg0").To("er0").
			AddE("ip_range").From("er0").To("r0").
			AddE("resource_link").From("uigp0").To("sg0").
			AddE("user_id_group_pairs").From("ir0").To("uigp0").
			AddE("ingress_rule").From("sg1").To("ir0").
			AddE("transient_resource_link").From("i0").To("sg1").
			AddE("egress_rule").From("sg1").To("er1").
			AddE("ip_range").From("er1").To("r1").
			Iterate()
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("error executing Gremlin query: %w", err)
	}

	return nil
}

func TestSetupMux_BlastRadius(t *testing.T) {
	if err := setupGraph(); err != nil {
		t.Fatalf("error setting up the initial graph: %v", err)
	}

	cfg := intel.Config{
		GremlinConfig: gremlin.Config{
			Endpoint: gremlinEndpoint,
			AuthMode: "plain",
		},
		ResolveTimeoutMs:     60000,
		BlastRadiusTimeoutMs: 60000,
	}
	mux, err := setupMux(cfg)
	if err != nil {
		t.Fatalf("could not set up mux: %v", err)
	}
	ts := httptest.NewServer(mux)
	defer ts.Close()

	url := ts.URL + "/v1/blast-radius?asset_type=IP&asset_identifier=1.2.3.4"
	res, err := http.Get(url)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: got=%v want=%v", res.StatusCode, http.StatusOK)
	}

	var got blastRadiusResp
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("malformed body: %v", err)
	}
	res.Body.Close()

	if diff := cmp.Diff(wantBlastRadiusResp, got); diff != "" {
		t.Errorf("responses mismatch (-want +got):\n%v", diff)
	}
}

func TestReadConfig(t *testing.T) {
	tests := []struct {
		name       string
		env        map[string]string
		wantConfig config
		wantNilErr bool
	}{
		{
			name: "set required config",
			env: map[string]string{
				"GREMLIN_ENDPOINT": "ws://127.0.0.1:8182/gremlin",
			},
			wantConfig: config{
				LogLevel:   defaultLogLevel,
				ListenAddr: defaultListenAddr,
				IntelConfig: intel.Config{
					GremlinConfig: gremlin.Config{
						Endpoint:      "ws://127.0.0.1:8182/gremlin",
						AuthMode:      defaultGremlinAuthMode,
						AWSRegion:     defaultAWSRegion,
						RetryLimit:    defaultGremlinRetryLimit,
						RetryDuration: defaultGremlinRetryDuration,
					},
					ResolveTimeoutMs:     defaultIntelResolveTimeoutMs,
					BlastRadiusTimeoutMs: defaultIntelBlastRadiusTimeoutMs,
				},
			},
			wantNilErr: true,
		},
		{
			name: "set optional config",
			env: map[string]string{
				"GREMLIN_ENDPOINT":              "ws://127.0.0.1:8182/gremlin",
				"LOG_LEVEL":                     "error",
				"LISTEN_ADDR":                   ":1234",
				"GREMLIN_AUTH_MODE":             "neptune_iam",
				"AWS_REGION":                    "eu-west-2",
				"GREMLIN_RETRY_LIMIT":           "10",
				"GREMLIN_RETRY_DURATION":        "10s",
				"INTEL_RESOLVE_TIMEOUT_MS":      "30000",
				"INTEL_BLAST_RADIUS_TIMEOUT_MS": "30000",
			},
			wantConfig: config{
				LogLevel:   "error",
				ListenAddr: ":1234",
				IntelConfig: intel.Config{
					GremlinConfig: gremlin.Config{
						Endpoint:      "ws://127.0.0.1:8182/gremlin",
						AuthMode:      "neptune_iam",
						AWSRegion:     "eu-west-2",
						RetryLimit:    10,
						RetryDuration: 10 * time.Second,
					},
					ResolveTimeoutMs:     30000,
					BlastRadiusTimeoutMs: 30000,
				},
			},
			wantNilErr: true,
		},
		{
			name:       "missing GREMLIN_ENDPOINT",
			env:        map[string]string{},
			wantConfig: config{},
			wantNilErr: false,
		},
		{
			name: "invalid GREMLIN_RETRY_DURATION",
			env: map[string]string{
				"GREMLIN_ENDPOINT":       "ws://127.0.0.1:8182/gremlin",
				"GREMLIN_RETRY_DURATION": "10x",
			},
			wantConfig: config{},
			wantNilErr: false,
		},
		{
			name: "zero GREMLIN_RETRY_DURATION",
			env: map[string]string{
				"GREMLIN_ENDPOINT":       "ws://127.0.0.1:8182/gremlin",
				"GREMLIN_RETRY_DURATION": "0",
			},
			wantConfig: config{
				LogLevel:   defaultLogLevel,
				ListenAddr: defaultListenAddr,
				IntelConfig: intel.Config{
					GremlinConfig: gremlin.Config{
						Endpoint:      "ws://127.0.0.1:8182/gremlin",
						AuthMode:      defaultGremlinAuthMode,
						AWSRegion:     defaultAWSRegion,
						RetryLimit:    defaultGremlinRetryLimit,
						RetryDuration: 0,
					},
					ResolveTimeoutMs:     defaultIntelResolveTimeoutMs,
					BlastRadiusTimeoutMs: defaultIntelBlastRadiusTimeoutMs,
				},
			},
			wantNilErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			config, err := readConfig()
			if (err == nil) != tt.wantNilErr {
				t.Errorf("unexpected error: wantNilErr=%v, got=%v", tt.wantNilErr, err)
			}

			if diff := cmp.Diff(tt.wantConfig, config); diff != "" {
				t.Errorf("config mismatch (-want +got):\n%v", diff)
			}
		})
	}
}
