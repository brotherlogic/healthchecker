package main

import (
	"fmt"

	"golang.org/x/net/context"
	"google.golang.org/grpc/status"

	dpb "github.com/brotherlogic/discovery/proto"
	gpb "github.com/brotherlogic/goserver/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	healthChecks = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "healthchecker_checks",
		Help: "The number of server requests",
	}, []string{"service", "identifier", "error"})
)

func (s *Server) checkHealth(ctx context.Context, server *dpb.RegistryEntry) error {
	conn, err := s.FDial(fmt.Sprintf("%v:%v", server.GetIdentifier(), server.GetPort()))
	if err != nil {
		return err
	}

	client := gpb.NewGoserverServiceClient(conn)
	alive, err := client.IsAlive(ctx, &gpb.Alive{})
	if err != nil {
		healthChecks.With(prometheus.Labels{
			"service":    server.GetName(),
			"identifier": server.GetIdentifier(),
			"error":      fmt.Sprintf("%v", status.Convert(err).Code()),
		}).Inc()
		return err
	}

	if alive.GetName() == server.GetName() {
		healthChecks.With(prometheus.Labels{
			"service":    server.GetName(),
			"identifier": server.GetIdentifier(),
			"error":      "nil",
		}).Inc()
		return nil
	}

	healthChecks.With(prometheus.Labels{
		"service":    server.GetName(),
		"identifier": server.GetIdentifier(),
		"error":      "unknown",
	}).Inc()
	return fmt.Errorf("Unable to determine if %v is alive -> %v, %v", server, alive, err)
}
