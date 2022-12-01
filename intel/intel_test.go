package intel

import (
	"errors"
	"testing"

	"github.com/adevinta/graph-intel-api/gremlin"

	gremlingo "github.com/apache/tinkerpop/gremlin-go/v3/driver"
	"github.com/google/go-cmp/cmp"
)

const gremlinEndpoint = "ws://127.0.0.1:8182/gremlin"

func TestAPIBlastRadius(t *testing.T) {
	gremlinConfig := gremlin.Config{
		Endpoint:   gremlinEndpoint,
		AuthMode:   "plain",
		RetryLimit: 1,
	}
	conn, err := gremlin.NewConnection(gremlinConfig)
	if err != nil {
		t.Fatalf("error creating Gremlin connection: %v", err)
	}

	_, err = conn.Query(func(g *gremlingo.GraphTraversalSource) ([]*gremlingo.Result, error) {
		err := <-g.
			AddV("Universe").Property("namespace", "altimeter").Property("version", 1).As("u0").
			AddV("altimeter_snapshot").Property("timestamp", 0).As("s0").
			AddV("ec2:network-interface").Property("public_ip", "1.2.3.4").Property("public_dns_name", "example.com").Property("status", "in-use").As("ni0").
			AddV("ec2:security-group").As("sg0").
			AddV("egress_rule").As("er0").
			AddV("ip_range").As("r0").
			AddV("user_id_group_pairs").As("uigp0").
			AddV("ingress_rule").As("ir0").
			AddV("ec2:security-group").As("sg1").
			AddV("ec2:instance").As("i0").
			AddV("egress_rule").As("er1").
			AddV("ip_range").As("r1").
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
		return nil, err
	})
	if err != nil {
		t.Fatalf("error executing Gremlin query: %v", err)
	}

	want := BlastRadiusResult{
		Score:    1234,
		Metadata: "net",
	}

	cfg := Config{
		GremlinConfig:        gremlinConfig,
		ResolveTimeoutMs:     60000,
		BlastRadiusTimeoutMs: 60000,
	}
	intelAPI, err := NewAPI(cfg)
	if err != nil {
		t.Fatalf("error creating intel API: %v", err)
	}

	got, err := intelAPI.BlastRadius("IP", "1.2.3.4")
	if err != nil {
		t.Fatalf("error calculating Blast Radius for IP: %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("scores mismatch for IP (-want +got):\n%v", diff)
	}

	got, err = intelAPI.BlastRadius("Hostname", "example.com")
	if err != nil {
		t.Fatalf("error calculating Blast Radius for Hostname: %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("scores mismatch for Hostname (-want +got):\n%v", diff)
	}
}

func TestAPIBlastRadius_NotFound(t *testing.T) {
	gremlinConfig := gremlin.Config{
		Endpoint:   gremlinEndpoint,
		AuthMode:   "plain",
		RetryLimit: 1,
	}
	conn, err := gremlin.NewConnection(gremlinConfig)
	if err != nil {
		t.Fatalf("error creating Gremlin connection: %v", err)
	}

	_, err = conn.Query(func(g *gremlingo.GraphTraversalSource) ([]*gremlingo.Result, error) {
		<-g.V().Drop().Iterate()
		return nil, nil
	})
	if err != nil {
		t.Fatalf("error executing Gremlin query: %v", err)
	}

	cfg := Config{
		GremlinConfig:        gremlinConfig,
		ResolveTimeoutMs:     60000,
		BlastRadiusTimeoutMs: 60000,
	}
	intelAPI, err := NewAPI(cfg)
	if err != nil {
		t.Fatalf("error creating intel API: %v", err)
	}

	_, err = intelAPI.BlastRadius("IP", "1.2.3.4")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("unexpected error: want=%v got=%v", ErrNotFound, err)
	}
}
