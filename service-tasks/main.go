package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"example.com/streaming-json/service-tasks/internal/export"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "generate:", err)
		os.Exit(1)
	}
}

func run() error {
	warehouse := flag.String("warehouse", "", "warehouse whose movements are exported")
	taskPlan := flag.String("task-plan", "", "task plan receiving the movements")
	flag.Parse()
	if *warehouse == "" || *taskPlan == "" {
		return errors.New("-warehouse and -task-plan are required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	store, err := export.NewStore(ctx, env("TASKS_DATABASE_URL", "postgres://demo:demo@localhost:5432/tasks"))
	if err != nil {
		return err
	}
	defer store.Close()
	if err = store.Reset(ctx, *taskPlan); err != nil {
		return err
	}

	start := time.Now()
	body, err := export.OpenStream(ctx, export.NewHTTPClient(), env("SERVICE_STOCKS_URL", "http://localhost:8080"), *warehouse)
	if err != nil {
		return err
	}
	defer body.Close()

	inserted, err := export.Run(ctx, body, store.InsertFunc(*taskPlan))
	if err != nil {
		return fmt.Errorf("after %d inserted movements: %w", inserted, err)
	}
	fmt.Printf("%d movements inserted into %s in %s\n", inserted, *taskPlan, time.Since(start).Round(time.Millisecond))
	return nil
}

func env(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok && value != "" {
		return value
	}
	return fallback
}
