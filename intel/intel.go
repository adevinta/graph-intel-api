// Package intel provides the API for getting all the intel information
// exposed by the Security Graph.
package intel

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/adevinta/graph-intel-api/gremlin"
	"github.com/adevinta/graph-intel-api/log"

	gremlingo "github.com/apache/tinkerpop/gremlin-go/v3/driver"
)

// maxQueryDepth is the maximum depth traversed in Gremlin queries.
const maxQueryDepth = int32(15)

// ErrNotFound is returned when an entity is not found.
var ErrNotFound = errors.New("not found")

// Config contains the configuration parameters.
type Config struct {
	// GremlinConfig is the Gremlin configuration parameters.
	GremlinConfig gremlin.Config

	// ResolveTimeoutMs is the query timeout in ms used when finding
	// assets. If zero, no timeout is set.
	ResolveTimeoutMs int

	// BlastRadiusTimeoutMs is the query timeout in ms used when
	// calculating the blast radius score. If zero, no timeout is set.
	BlastRadiusTimeoutMs int
}

// API implements the Intel API of the Security Graph.
type API struct {
	cfg      Config
	conn     gremlin.Connection
	resolver *net.Resolver
}

// NewAPI creates a new intel API using the given config.
func NewAPI(cfg Config) (API, error) {
	conn, err := gremlin.NewConnection(cfg.GremlinConfig)
	if err != nil {
		return API{}, fmt.Errorf("could not create a Gremlin connection: %w", err)
	}

	api := API{
		conn:     conn,
		resolver: &net.Resolver{PreferGo: true},
	}
	return api, nil
}

// BlastRadiusResult represents the result of calculating the blast radius
// score for a given asset.
type BlastRadiusResult struct {
	// Score contains the blast radius score for a given asset.
	Score float64

	// Metadata contains information about how a blast radius was
	// calculated.
	Metadata string
}

// BlastRadius returns the blast radius of a given asset. It returns a
// [BlastRadiusResult] with the score and the metadata about how score was
// calculated.
func (api API) BlastRadius(typ, identifier string) (BlastRadiusResult, error) {
	vid, err := api.resolveAsset(typ, identifier)
	if err != nil {
		return BlastRadiusResult{}, fmt.Errorf("could not resolve asset: %w", err)
	}

	score, err := api.netBlastRadius(vid)
	if err != nil {
		return BlastRadiusResult{}, fmt.Errorf("could not calculate net blast radius: %w", err)
	}

	result := BlastRadiusResult{
		Score:    score,
		Metadata: `net`,
	}

	return result, nil
}

// resolveAsset returns the vertex ID of an asset identified by its type and
// identifier.
func (api API) resolveAsset(typ, identifier string) (vid string, err error) {
	switch typ {
	case "IP":
		return api.resolveIP(identifier)
	case "Hostname":
		vid, err = api.resolveHostname(identifier)
		if err == nil {
			return vid, nil
		}

		log.Debug.Printf("could not find hostname %q: fallback to DNS lookup", identifier)

		ips, err := api.resolver.LookupHost(context.Background(), identifier)
		if err != nil {
			return "", fmt.Errorf("DNS lookup error for %q: %w", identifier, err)
		}

		for _, ip := range ips {
			vid, err = api.resolveIP(ip)
			if err == nil {
				break
			}
		}
		return vid, err
	default:
		return "", fmt.Errorf("unsupported asset type: %v", typ)
	}
}

// resolveHostname returns de vertex ID of a given hostname.
func (api API) resolveHostname(hostname string) (vid string, err error) {
	results, err := api.conn.Query(func(g *gremlingo.GraphTraversalSource) ([]*gremlingo.Result, error) {
		t := g

		if api.cfg.ResolveTimeoutMs > 0 {
			t = t.With("evaluationTimeout", api.cfg.ResolveTimeoutMs)
		}

		return t.
			V().
			HasLabel(
				"ec2:instance",
				"ec2:network-interface",
				"elbv1:loadbalancer",
				"elbv2:loadbalancer",
			).
			Union(
				gremlingo.T__.Has("private_dns_name", hostname).HasLabel("ec2:instance"),
				gremlingo.T__.Has("public_dns_name", hostname).HasLabel("ec2:instance"),

				gremlingo.T__.Has("private_dns_name", hostname).HasLabel("ec2:network-interface").Has("status", "in-use"),
				gremlingo.T__.Has("public_dns_name", hostname).HasLabel("ec2:network-interface").Has("status", "in-use"),

				gremlingo.T__.Has("dns_name", hostname).HasLabel("elbv1:loadbalancer"),

				gremlingo.T__.Has("dns_name", hostname).HasLabel("elbv2:loadbalancer"),
			).
			As("assets").
			In("includes").HasLabel("altimeter_snapshot").As("snapshots").
			In("universe_of").HasLabel("Universe").Has("namespace", "altimeter").Has("version", 1).
			Select("assets").
			Order().By(gremlingo.T__.Select("snapshots").Values("timestamp"), gremlingo.Order.Desc).
			Limit(1).
			Id().
			ToList()
	})
	if err != nil {
		return "", fmt.Errorf("query error: %w", err)
	}

	if len(results) == 0 {
		return "", ErrNotFound
	}

	return results[0].GetString(), nil
}

// resolveIP returns de vertex ID of a given IP.
func (api API) resolveIP(ip string) (vid string, err error) {
	results, err := api.conn.Query(func(g *gremlingo.GraphTraversalSource) ([]*gremlingo.Result, error) {
		t := g

		if api.cfg.ResolveTimeoutMs > 0 {
			t = t.With("evaluationTimeout", api.cfg.ResolveTimeoutMs)
		}

		return t.
			V().
			HasLabel("ec2:network-interface").
			Has("status", "in-use").
			Or(
				gremlingo.T__.Has("public_ip", ip),
				gremlingo.T__.Has("private_ip_address", ip),
			).
			As("assets").
			In("includes").HasLabel("altimeter_snapshot").As("snapshots").
			In("universe_of").HasLabel("Universe").Has("namespace", "altimeter").Has("version", 1).
			Select("assets").
			Order().By(gremlingo.T__.Select("snapshots").Values("timestamp"), gremlingo.Order.Desc).
			Limit(1).
			Id().
			ToList()
	})
	if err != nil {
		return "", fmt.Errorf("query error: %w", err)
	}

	if len(results) == 0 {
		return "", ErrNotFound
	}

	return results[0].GetString(), nil
}

// netBlastRadius returns the Network Blast Radius score calculated from a
// vertex ID.
func (api API) netBlastRadius(vid string) (float64, error) {
	results, err := api.conn.Query(func(g *gremlingo.GraphTraversalSource) ([]*gremlingo.Result, error) {
		t := g

		if api.cfg.BlastRadiusTimeoutMs > 0 {
			t = t.With("evaluationTimeout", api.cfg.BlastRadiusTimeoutMs)
		}

		return t.
			V(vid).
			Union(
				gremlingo.T__.OutE("resource_link").InV(),
				gremlingo.T__.OutE("transient_resource_link").InV(),
			).
			HasLabel("ec2:security-group").
			Repeat(
				gremlingo.T__.
					Union(
						gremlingo.T__.
							OutE("egress_rule").InV().HasLabel("egress_rule").
							OutE("ip_range").InV().HasLabel("ip_range"),
						gremlingo.T__.
							InE("resource_link").OutV().HasLabel("user_id_group_pairs").
							InE("user_id_group_pairs").OutV().HasLabel("ingress_rule").
							InE("ingress_rule").OutV().HasLabel("ec2:security-group").
							Union(
								gremlingo.T__.Identity(),
								gremlingo.T__.InE().OutV().Or(
									gremlingo.T__.HasLabel("ec2:instance"),
									gremlingo.T__.HasLabel("elb:loadbalancer"),
									gremlingo.T__.HasLabel("elbv2:loadbalancer"),
									gremlingo.T__.HasLabel("rds:db"),
								),
							),
					).
					SimplePath(),
			).
			Times(maxQueryDepth).
			Emit().
			Project("id", "label", "steps").
			By(gremlingo.T__.Id()).
			By(gremlingo.T__.Label()).
			By(gremlingo.T__.Path().Count(gremlingo.Scope.Local)).
			ToList()
	})
	if err != nil {
		return 0, fmt.Errorf("query error: %w", err)
	}

	score := 0.0
	for _, result := range results {
		rsc, err := parseResource(result)
		if err != nil {
			return 0, fmt.Errorf("invalid result: %w", err)
		}

		weight := 1.0
		if rsc.label == "ec2:security-group" {
			weight = 0.0
		}
		score += (1.0 / rsc.steps) * weight
	}

	return score, nil
}

// resource represents a parsed result of the Blast Radius query.
type resource struct {
	id    string
	label string
	steps float64
}

// parseResource parses a Gremlin result returned by the Blast Radius query.
func parseResource(result *gremlingo.Result) (resource, error) {
	obj := result.GetInterface()

	m, ok := obj.(map[any]any)
	if !ok {
		return resource{}, errors.New("invalid result type")
	}

	var r resource

	for k, v := range m {
		sk, ok := k.(string)
		if !ok {
			return resource{}, errors.New("key is not a string")
		}

		switch sk {
		case "id":
			id, ok := v.(string)
			if !ok {
				return resource{}, errors.New("id is not a string")
			}
			r.id = id
		case "label":
			label, ok := v.(string)
			if !ok {
				return resource{}, errors.New("label is not a string")
			}
			r.label = label
		case "steps":
			steps, ok := v.(int64)
			if !ok {
				return resource{}, errors.New("steps is not an int64")
			}
			r.steps = float64(steps)
		default:
			return resource{}, fmt.Errorf("unknown key %q", sk)
		}
	}

	return r, nil
}
