package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/MaxPa1/go-metrics/internal/agent"
	"github.com/MaxPa1/go-metrics/internal/config"

	"github.com/go-resty/resty/v2"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadAgentConfig()
	if err != nil {
		log.Fatal(err)
	}

	client := resty.New().
		SetTimeout(5 * time.Second)

	agent.NewMetricAgent(cfg, client).Run(ctx)
	log.Println("Agent stopped")
}
