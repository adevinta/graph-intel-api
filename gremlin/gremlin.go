// Package gremlin provides access to a Gremlin server.
package gremlin

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/adevinta/graph-intel-api/log"

	gremlingo "github.com/apache/tinkerpop/gremlin-go/v3/driver"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
)

// emptyStringSHA256 is the hex encoded sha256 value of an empty string.
const emptyStringSHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// ErrNotFound is returned when a Gremlin query times out.
var ErrTimeout = errors.New("timeout error")

// Config contains the configuration parameters needed to interact with a
// Gremlin server.
type Config struct {
	// Endpoint is the Gremlin Endpoint.
	Endpoint string

	// AuthMode is the authentication mode. Valid values: "plain",
	// "neptune_iam".
	AuthMode string

	// AWSRegion is the AWS region in case of using Neptune.
	AWSRegion string

	// RetryLimit is the number of retries before returning error.
	RetryLimit int

	// RetryDuration is the time to wait between retries.
	RetryDuration time.Duration
}

// connHandler is called to create a connection with a Gremlin server.
type connHandler func(cfg Config) (*gremlingo.DriverRemoteConnection, error)

// A Connection handles the connection with the Gremlin server. This includes
// authentication, reconnections and retries.
type Connection struct {
	cfg Config
	h   connHandler
}

// Connection creates a [Connection] with the provided configuration.
func NewConnection(cfg Config) (Connection, error) {
	var connHandler connHandler

	switch cfg.AuthMode {
	case "plain":
		connHandler = connectPlain
	case "neptune_iam":
		connHandler = connectNeptuneIam
	default:
		return Connection{}, errors.New("invalid auth mode")
	}

	conn := Connection{
		cfg: cfg,
		h:   connHandler,
	}
	return conn, nil
}

// connectNeptuneIam is a [connHandler] for Neptune that creates an
// authenticated connection using IAM.
func connectNeptuneIam(cfg Config) (*gremlingo.DriverRemoteConnection, error) {
	auth, err := getNeptuneAuth(context.Background(), cfg.Endpoint, cfg.AWSRegion)
	if err != nil {
		return nil, fmt.Errorf("error getting AWS auth headers: %v", err)
	}

	log.Debug.Printf("connecting to Neptune")
	conn, err := gremlingo.NewDriverRemoteConnection(cfg.Endpoint, func(settings *gremlingo.DriverRemoteConnectionSettings) {
		settings.AuthInfo = gremlingo.HeaderAuthInfo(auth)
		settings.LogVerbosity = gremlingo.Off
	})
	return conn, err
}

// connectPlain is a [connHandler] for Gremlin server that creates an
// unauthenticated connection.
func connectPlain(cfg Config) (*gremlingo.DriverRemoteConnection, error) {
	log.Debug.Printf("connecting to Gremlin server")
	conn, err := gremlingo.NewDriverRemoteConnection(cfg.Endpoint, func(settings *gremlingo.DriverRemoteConnectionSettings) {
		settings.LogVerbosity = gremlingo.Off
	})
	return conn, err
}

// getNeptuneAuth returns the AWS auth headers required to interact with
// Neptune.
func getNeptuneAuth(ctx context.Context, endpoint, region string) (http.Header, error) {
	log.Debug.Printf("getting Neptune auth")

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not load AWS config: %w", err)
	}

	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get AWS credentials: %w", err)
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, creds, req, emptyStringSHA256, "neptune-db", region, time.Now()); err != nil {
		return nil, fmt.Errorf("could not sign request: %w", err)
	}

	return req.Header, nil
}

// QueryFunc represents a Gremlin query in the context of a [Connection]. It is
// executed by [Connectino.Query].
type QueryFunc func(*gremlingo.GraphTraversalSource) ([]*gremlingo.Result, error)

// Query executes cf taking care of the authentication, reconnections and
// retries.
func (conn Connection) Query(cf QueryFunc) ([]*gremlingo.Result, error) {
	for i := 0; i < conn.cfg.RetryLimit; i++ {
		results, err := conn.execQuery(cf)
		if err == nil {
			return results, nil
		}

		log.Debug.Printf("error executing query (%v/%v): %v", i+1, conn.cfg.RetryLimit, err)

		if strings.Contains(err.Error(), `"code":"TimeLimitExceededException"`) {
			return nil, ErrTimeout
		}

		if i < conn.cfg.RetryLimit-1 {
			jitter := time.Duration(rand.Int63n(1000)) * time.Millisecond
			t := conn.cfg.RetryDuration + jitter

			log.Debug.Printf("retrying in %v", t)
			time.Sleep(t)
		}
	}

	return nil, errors.New("max retries exceeded")
}

// execQuery executes cf in the context of a new remote Gremlin connection.
func (conn Connection) execQuery(cf QueryFunc) ([]*gremlingo.Result, error) {
	rc, err := conn.h(conn.cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating driver remote connection: %v", err)
	}
	defer rc.Close()

	g := gremlingo.Traversal_().WithRemote(rc)
	return cf(g)
}
