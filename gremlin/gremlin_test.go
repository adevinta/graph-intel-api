package gremlin

import (
	"fmt"
	"testing"

	gremlingo "github.com/apache/tinkerpop/gremlin-go/v3/driver"
	"github.com/google/go-cmp/cmp"
)

const gremlinEndpoint = "ws://127.0.0.1:8182/gremlin"

var wantVertices = []string{"v1", "v2"}

func setupGraph() error {
	conn, err := gremlingo.NewDriverRemoteConnection(gremlinEndpoint, func(settings *gremlingo.DriverRemoteConnectionSettings) {
		settings.LogVerbosity = gremlingo.Off
	})
	if err != nil {
		return fmt.Errorf("could not connect to gremlin-server: %w", err)
	}
	defer conn.Close()

	g := gremlingo.Traversal_().WithRemote(conn)

	<-g.V().Drop().Iterate()

	for _, id := range wantVertices {
		<-g.AddV(id).Iterate()
	}

	return nil
}

func TestNewConnection_InvalidAuthMode(t *testing.T) {
	cfg := Config{AuthMode: "invalid"}
	if _, err := NewConnection(cfg); err == nil {
		t.Error("unexpected nil error")
	}
}

func TestConnectionQuery(t *testing.T) {
	if err := setupGraph(); err != nil {
		t.Fatalf("error setting up graph: %v", err)
	}

	cfg := Config{
		Endpoint:   gremlinEndpoint,
		AuthMode:   "plain",
		RetryLimit: 1,
	}
	conn, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("error creating connection: %v", err)
	}

	results, err := conn.Query(func(g *gremlingo.GraphTraversalSource) ([]*gremlingo.Result, error) {
		return g.V().Label().ToList()
	})
	if err != nil {
		t.Fatalf("query error: %v", err)
	}

	var got []string
	for _, r := range results {
		got = append(got, r.GetString())
	}

	if diff := cmp.Diff(wantVertices, got); diff != "" {
		t.Errorf("vertices mismatch (-want +got):\n%v", diff)
	}
}
