package main

import (
	"fmt"

	"golang.org/x/net/context"

	dpb "github.com/brotherlogic/discovery/proto"
	gpb "github.com/brotherlogic/goserver/proto"
)

func (s *Server) checkHealth(ctx context.Context, server *dpb.RegistryEntry) error {
	conn, err := s.FDial(fmt.Sprintf("%v:%v", server.GetIdentifier(), server.GetPort()))
	if err != nil {
		return err
	}

	client := gpb.NewGoserverServiceClient(conn)
	alive, err := client.IsAlive(ctx, &gpb.Alive{})
	if err != nil {
		return err
	}

	if alive.GetName() == server.GetName() {
		return nil
	}

	return fmt.Errorf("Unable to determine if %v is alive -> %v, %v", server, alive, err)
}
