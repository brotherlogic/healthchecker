package main

import (
	"fmt"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc/status"

	dpb "github.com/brotherlogic/discovery/proto"
	gpb "github.com/brotherlogic/goserver/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	pb "github.com/brotherlogic/healthchecker/proto"
)

var (
	healthChecks = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "healthchecker_checks",
		Help: "The number of server requests",
	}, []string{"service", "identifier", "error"})
)

func (s *Server) recordMetrics(config *pb.Config) {

}

func (s *Server) runCheck(ctx context.Context, config *pb.Config) {
	defer s.recordMetrics(config)

	var best *pb.Check
	last := time.Now().Unix()
	for _, check := range config.GetChecks() {
		if check.LastCheck < last {
			best = check
			last = check.LastCheck
		}
	}

	if best != nil {
		err := s.checkHealth(ctx, best.GetEntry())
		best.LastCheck = time.Now().Unix()
		if err == nil {
			best.LastGoodCheck = best.LastCheck
			best.BadChecksSinceLastGood = 0
		} else {
			best.BadChecksSinceLastGood++
		}
	}
}

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

	return fmt.Errorf("unable to determine if %v is alive -> %v, %v", server, alive, err)
}
