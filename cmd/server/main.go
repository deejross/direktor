package main

import (
	"github.com/deejross/direktor/internal/server"
	"github.com/deejross/direktor/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	log := logger.New("cmd/server")

	if err := server.Start(); err != nil {
		log.Fatal("could not start server", zap.Error(err))
	}
}
