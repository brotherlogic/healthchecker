package main

import (
	"fmt"
	"time"

	"github.com/brotherlogic/goserver"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	dpb "github.com/brotherlogic/discovery/proto"
	pbg "github.com/brotherlogic/goserver/proto"
	"github.com/brotherlogic/goserver/utils"
)

//Server main server type
type Server struct {
	*goserver.GoServer
}

// Init builds the server
func Init() *Server {
	s := &Server{
		GoServer: &goserver.GoServer{},
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

func main() {
	server := Init()
	server.PrepServer()
	server.Register = server

	err := server.RegisterServerV2("healthchecker", false, true)
	if err != nil {
		return
	}

	ctx, cancel := utils.ManualContext("healthchecker-init", time.Minute)
	s.Log(fmt.Sprintf("HEALTH CHECK: %v", server.checkHealth(ctx, &dpb.RegistryEntry{Identifier: "dev", Port: int32(50055), Name: "discovery"})))
	cancel()

	server.Serve()
}
