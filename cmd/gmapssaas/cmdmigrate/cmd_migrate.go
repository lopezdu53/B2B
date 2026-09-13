package cmdmigrate

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	gocli "github.com/gosom/google-maps-scraper/cli"
	"github.com/gosom/google-maps-scraper/env"
	"github.com/gosom/google-maps-scraper/migrations"
	saas "github.com/gosom/google-maps-scraper/saas"
)

// Command applies all pending database migrations. It is meant to be run once
// before starting the server (e.g. as a one-off task or a container startup
// step in Docker / Easypanel / Kubernetes deployments).
var Command = &cli.Command{
	Name:  "migrate",
	Usage: "Apply pending database migrations",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "database-url",
			Usage:   "PostgreSQL connection string",
			Value:   "postgres://postgres:postgres@localhost:5432/gmaps_pro?sslmode=disable",
			Sources: cli.EnvVars(saas.EnvDatabaseURL),
		},
	},
	Action: runMigrate,
}

func runMigrate(_ context.Context, cmd *cli.Command) error {
	gocli.PrintBanner("Google Maps Scraper Pro - Database Migrations")

	databaseURL := cmd.String("database-url")

	env.LogUnsetEnvs(saas.EnvDatabaseURL)

	fmt.Println("Running database migrations...")

	n, err := migrations.RunWithDSN(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	fmt.Printf("Applied %d migration(s).\n", n)

	return nil
}
