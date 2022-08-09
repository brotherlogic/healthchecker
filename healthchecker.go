package main

import (
	"fmt"
	"time"

	"github.com/brotherlogic/goserver"
	pb "github.com/brotherlogic/healthchecker/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	dpb "github.com/brotherlogic/discovery/proto"
	pbg "github.com/brotherlogic/goserver/proto"
	"github.com/brotherlogic/goserver/utils"
)

var (
	tracked = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "healthchecker_tracked",
		Help: "The number of server requests",
	})
)

//Server main server type
type Server struct {
	*goserver.GoServer
	config   *pb.Config
	lastPull time.Time
}

// Init builds the server
func Init() *Server {
	s := &Server{
		GoServer: &goserver.GoServer{},
		config:   &pb.Config{},
		lastPull: time.Now().Add(-time.Hour),
	}
	return s
}

// DoRegister does RPC registration
func (s *Server) DoRegister(server *grpc.Server) {

}

// ReportHealth alerts if we're not healthy
func (s *Server) ReportHealth() bool {
	return true
}

// Shutdown the server
func (s *Server) Shutdown(ctx context.Context) error {
	return nil
}

// Mote promotes/demotes this server
func (s *Server) Mote(ctx context.Context, master bool) error {
	return nil
}

// GetState gets the state of the server
func (s *Server) GetState() []*pbg.State {
	return []*pbg.State{}
}

func (s *Server) buildConfig(ctx context.Context, config *pb.Config) error {
	conn, err := s.FDialServer(ctx, "discovery")
	if err != nil {
		return err
	}
	defer conn.Close()

	client := dpb.NewDiscoveryServiceV2Client(conn)

	all, err := client.Get(ctx, &dpb.GetRequest{})
	if err != nil {
		return err
	}

	for _, service := range all.GetServices() {
		found := false
		for _, entry := range config.GetChecks() {
			if entry.GetEntry().GetIdentifier() == service.Identifier && entry.GetEntry().GetName() == service.GetName() {
				found = true
			}
		}

		if !found {
			config.Checks = append(config.Checks, &pb.Check{Entry: service})
		}
	}

	newAll := []*pb.Check{}
	for _, entry := range config.GetChecks() {
		found := false
		for _, service := range all.GetServices() {
			if entry.GetEntry().GetIdentifier() == service.Identifier && entry.GetEntry().GetName() == service.GetName() {
				found = true
			}
		}

		if found {
			newAll = append(newAll, entry)
		}
	}

	config.Checks = newAll

	tracked.Set(float64(len(config.Checks)))
	return nil
}

func (s *Server) runHealthCheck() {
	for !s.LameDuck {
		ctx, cancel := utils.ManualContext("healthcheck-loop", time.Minute)
		if time.Since(s.lastPull) > time.Hour {
			err := s.buildConfig(ctx, s.config)
			if err == nil {
				s.lastPull = time.Now()
			} else {
				s.Log(fmt.Sprintf("Unable to read config: %v", err))
			}
		}

		s.runCheck(ctx, s.config)
		cancel()

		time.Sleep(time.Second)
	}
}

func main() {
	server := Init()
	server.PrepServer("healthchecker")
	server.Register = server

	err := server.RegisterServerV2(false)
	if err != nil {
		return
	}

	go server.runHealthCheck()

	server.Serve()
}
