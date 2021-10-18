package main

import (
	dpb "github.com/brotherlogic/discovery/proto"
	"golang.org/x/net/context"
)

func (s *Server) checkHealth(ctx context.Context, server *dpb.RegistryEntry) error {
	return nil
}
