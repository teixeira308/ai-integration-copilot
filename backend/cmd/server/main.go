package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/guilhermeteixeira/ai-integration-copilot/backend/internal/api"
	"github.com/guilhermeteixeira/ai-integration-copilot/backend/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	api.RegisterRoutes(router, cfg)

	listenAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("starting backend on %s", listenAddr)

	if err := router.Run(listenAddr); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
