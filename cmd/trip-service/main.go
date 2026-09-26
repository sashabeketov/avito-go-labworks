package main

import (
	"context"
	"fmt"

	"github.com/sashabeketov/avito-go-labworks/internal/config"
	"github.com/sashabeketov/avito-go-labworks/internal/storage/postgres"
)

func main() {
	err := run(context.Background())
	if err != nil {
		fmt.Println(err)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	pool, err := postgres.CreateNewPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("open PostgreSQL: %w", err)
	}
	defer pool.Close()

	fmt.Println("NICE")
	return nil
}
