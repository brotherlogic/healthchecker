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

	totalChecks = promauto.NewCounter(prometheus.CounterOpts{
		Name: "healthchecker_totalchecks",
		Help: "The number of server requests",
	})

	lastChecked = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "healthchecker_lastChecked",
		Help: "The number of serverrequests",
	})

	healthErrors = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "healthchecker_errors",
		Help: "The number of server requests",
	}, []string{"service", "identifier"})
)

func (s *Server) recordMetrics(config *pb.Config) {
	best := time.Now().Unix()
	for _, check := range config.GetChecks() {
		if check.LastCheck < best {
			best = check.LastCheck
		}
	}

	lastChecked.Set(float64(best))
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
		if err != nil {
			s.CtxLog(ctx, fmt.Sprintf("Error checking %v -> %v", best.GetEntry(), err))
		}

		best.LastCheck = time.Now().Unix()
		if err == nil {
			best.LastGoodCheck = best.LastCheck
			best.BadChecksSinceLastGood = 0
			healthErrors.With(prometheus.Labels{"service": best.Entry.Name, "identifier": best.Entry.Identifier}).Set(float64(0))
		} else {
			best.BadChecksSinceLastGood++
			healthErrors.With(prometheus.Labels{"service": best.Entry.Name, "identifier": best.Entry.Identifier}).Set(float64(best.BadChecksSinceLastGood))
			if best.BadChecksSinceLastGood > 5 {
				s.CtxLog(ctx, fmt.Sprintf("Removing %v: most recent error %v", best, err))

				err := s.unregister(ctx, best.GetEntry())
				if err == nil {
					var nchecks []*pb.Check
					for _, check := range config.GetChecks() {
						if check.Entry.Identifier != best.Entry.Identifier ||
							check.Entry.Name != best.Entry.Name {
							nchecks = append(nchecks, check)
						}
					}
					config.Checks = nchecks
				}
			}
		}
	}
}

func (s *Server) unregister(ctx context.Context, entry *dpb.RegistryEntry) error {
	conn, err := s.FDialServer(ctx, "discovery")
	if err != nil {
		return err
	}
	defer conn.Close()

	client := dpb.NewDiscoveryServiceV2Client(conn)
	_, err = client.Unregister(ctx, &dpb.UnregisterRequest{
		Reason:  "healthcheck-unregister",
		Service: &dpb.RegistryEntry{Identifier: entry.GetIdentifier(), Name: entry.GetName()},
	})

	return err
}

func (s *Server) checkHealth(ctx context.Context, server *dpb.RegistryEntry) error {
	totalChecks.Inc()

	conn, err := s.FDial(fmt.Sprintf("%v:%v", server.GetIp(), server.GetPort()))
	if err != nil {
		return err
	}
	defer conn.Close()

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
