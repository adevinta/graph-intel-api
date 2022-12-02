package intel

import (
	"fmt"
	"testing"

	"github.com/adevinta/graph-intel-api/gremlin"

	gremlingo "github.com/apache/tinkerpop/gremlin-go/v3/driver"
	"github.com/google/go-cmp/cmp"
)

const gremlinEndpoint = "ws://127.0.0.1:8182/gremlin"

var wantBlastRadiusResult = BlastRadiusResult{
	Score:    0.3106893106893107,
	Metadata: "net",
}

func setupBlastRadiusGraph() error {
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

func TestAPIBlastRadius(t *testing.T) {
	tests := []struct {
		name       string
		typ        string
		identifier string
		wantNilErr bool
	}{
		{
			name:       "IP",
			typ:        "IP",
			identifier: "1.2.3.4",
			wantNilErr: true,
		},
		{
			name:       "Hostname",
			typ:        "Hostname",
			identifier: "example.com",
			wantNilErr: true,
		},
		{
			name:       "not found",
			typ:        "Hostname",
			identifier: "unknown",
			wantNilErr: false,
		},
		{
			name:       "invalid type",
			typ:        "inunknown",
			identifier: "1.2.3.4",
			wantNilErr: false,
		},
	}

	if err := setupBlastRadiusGraph(); err != nil {
		t.Fatalf("error setting up the initial graph: %v", err)
	}

	cfg := Config{
		GremlinConfig: gremlin.Config{
			Endpoint: gremlinEndpoint,
			AuthMode: "plain",
		},
		ResolveTimeoutMs:     60000,
		BlastRadiusTimeoutMs: 60000,
	}
	intelAPI, err := NewAPI(cfg)
	if err != nil {
		t.Fatalf("error creating intel API: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := intelAPI.BlastRadius(tt.typ, tt.identifier)

			if (err == nil) != tt.wantNilErr {
				t.Fatalf("unexpected error: wantNilErr=%v, got=%v", tt.wantNilErr, err)
			}

			if err != nil {
				return
			}

			if diff := cmp.Diff(wantBlastRadiusResult, got); diff != "" {
				t.Errorf("Blast Radius scores mismatch (-want +got):\n%v", diff)
			}
		})
	}
}
